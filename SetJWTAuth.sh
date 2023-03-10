#!/bin/bash

# 建立設定檔案的檔案變數
FILE_REDIS_ACL="./RedisACLCluster/users.acl"
FILE_JWTAUTH_ENV="./JWTAuthSvr/.env"
FILE_HAPROXY_CONF="./RedisACLCluster/haproxy/conf/haproxy.cfg"

# 設立四個 postgresql 密碼環境變數
PG_SUPER_PASS="#JWTAuth1234#"
PG_ADMIN_PASS="#JWTAuth1234#"
PG_OP_PASS="#JWTAuth1234#"
PG_QRY_PASS="#JWTAuth1234#"

# 設定 redis cluster 中會用到的密碼環境變數
REDIS_OP_PASS="#JWTAuth1234#"
REDIS_READER_PASS="#JWTAuth1234#"
REDIS_REP_PASS="#JWTAuth1234#"
REDIS_MASTER_AUTH_PASS="#JWTAuth1234#"
HAPROXY_AUTH_PASS="#JWTAuth1234#"

# 設定 JWTAuth 安全參數
JWT_SEC_KEY="696ceb369e628963ddd6e17ba4acc76c9a812d19fbfaad68d58581ca513e76e0"
JWT_AUTH_IP_OR_FQDN="1.2.3.4"
JWT_AUTH_PORT="20001"

# 設定系統使用的時區
SYS_TZONE="Asia/Taipei"

# 設定 replica nodes 的數目
REPLICA_NUM=3

# 設定 sentinel nodes 的數目，必須是奇數
SENTINEL_NUM=5

# 設定 sentinel quorum 的數目為總 sentinels 數目的一半以上
ST_QUORUM_NUM=$(echo "($SENTINEL_NUM + 1)/2" | bc)

# 刪除之前的 users.acl 並且複製模板產生 users.acl
rm -f "$FILE_REDIS_ACL"
cat ./RedisACLCluster/users.acl.template > "$FILE_REDIS_ACL"

# 設定 redis acl file 的密碼
sed -e 's/{RedisOpPass}/'"$REDIS_OP_PASS"'/g' -i "$FILE_REDIS_ACL"
sed -e 's/{RedisReaderPass}/'"$REDIS_READER_PASS"'/g' -i "$FILE_REDIS_ACL"
sed -e 's/{RedisRepPass}/'"$REDIS_REP_PASS"'/g' -i "$FILE_REDIS_ACL"
sed -e 's/{RedisSentPass}/'"$REDIS_MASTER_AUTH_PASS"'/g' -i "$FILE_REDIS_ACL"
sed -e 's/{HaproxyPass}/'"$HAPROXY_AUTH_PASS"'/g' -i "$FILE_REDIS_ACL"

chmod 400 "$FILE_REDIS_ACL"

# 刪除之前的 JWTAuth 使用的 .env 並且複製模板產生 .env
rm -f "$FILE_JWTAUTH_ENV"
cat "./RedisACLCluster/users.acl.template" > "$FILE_JWTAUTH_ENV"

# 設定 JWTAuth .env 秘密參數
sed -e 's/{RedisOpPass}/'"$REDIS_OP_PASS"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{RedisReaderPass}/'"$REDIS_READER_PASS"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{PgQryPass}/'"$PG_QRY_PASS"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{PgOpPass}/'"$PG_OP_PASS"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{JwtSecKey}/'"$JWT_SEC_KEY"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{PassSalt}/'"$USER_PASS_SALT"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{AuthSvrIp}/'"$JWT_AUTH_IP_OR_FQDN"'/g' -i "$FILE_JWTAUTH_ENV"
sed -e 's/{AuthSvrPort}/'"$JWT_AUTH_PORT"'/g' -i "$FILE_JWTAUTH_ENV"

chmod 400 "$FILE_JWTAUTH_ENV"

# 如果是外網，可以使用 docker pull 取得官方的 images ，然後需要對 alpine linux 做 WORKDIR 的設定，然後重新 commit 成 alpine_env:latest 。
docker pull postgres
docker pull redis
docker pull haproxy:latest
docker pull alpine

# 抓取 GO 編譯環境
docker pull golang:alpine

# 編譯 JWTAuth
docker run --rm -v ./src:/usr/src -w /usr/src/JWTAuth golang:alpine go build -v
chown jwtauth:jwtauth ./src/JWTAuth/JWTAuth
chmod 500 ./src/JWTAuth/JWTAuth
mv ./src/JWTAuth/JWTAuth ./JWTAuthSvr/

# 建立 GO 執行環境的映像檔
# 修改 alpine image ，增加 WORKDIR 然後重新 build 成新的 image alpine_env:latest
# mkdir ./tmp && echo -e "FROM alpine\nRUN mkdir /app\nWORKDIR /app" | docker build -t alpine_env:latest -f- ./tmp && rm -rf ./tmp && docker rmi alpine:latest

# 建立網路
docker network create JwtNet
docker network create RedisACLNet

# 建立資料夾給 postgresql 資料庫容器 mount 使用。
mkdir ./ForPgUserAuth
chown jwtauth:jwtauth ./ForPgUserAuth

# 建立 PgUserAuth
docker run -itd \
    --network JwtNet \
    --name PgUserAuth \
    -h PgUserAuth \
    -p 25432:5432 \
    -e POSTGRES_PASSWORD=$PG_SUPER_PASS \
    -e PASS_ADMIN=$PG_ADMIN_PASS \
    -e PASS_OP=$PG_OP_PASS \
    -e PASS_QRY=$PG_QRY_PASS \
    -v ./ForPgUserAuth:/var/lib/postgresql/data \
    postgres:latest

# 等容器建好
# [ $(docker logs PgUserAuth | grep 'listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432' | wc -l) -lt 2 ] 是一個條件語句，
# 它的作用是執行一個 Docker 容器，使用 docker logs 命令檢查容器日誌，並使用 grep 命令過濾出特定的字串，
# 最後使用 wc -l 命令計算符合條件的行數。如果符合條件的行數少於 2 行，條件為真。
while [ $(docker logs PgUserAuth | grep 'listening on Unix socket "/var/run/postgresql/.s.PGSQL.5432' | wc -l) -lt 2 ]
do
    sleep 1s
done

echo "PgUserAuth is up..."

# 修正資料庫時區，因為時區變數有 / ，所以使用 @ 取代 /
# sed -e 's/Etc\/UTC/Asia\/Taipei/g' -i ./ForPgUserAuth/postgresql.conf
sed -e 's@Etc\/UTC@'"$SYS_TZONE"'@g' -i ./ForPgUserAuth/postgresql.conf

# 把要執行的 SQL 複製到容器 binding 的目錄下
cp ./*.sql ./ForPgUserAuth
cp ./SetPgUserauth.sh ./ForPgUserAuth

# 重載系統設定
docker exec -d -u postgres PgUserAuth pg_ctl reload

# 建立使用者群、grant 預設的物件權限、在 PgUserAuth 中塞入預先定義好的 schema 。注意，這裡要"容器內"絕對路徑。
docker exec -it PgUserAuth /var/lib/postgresql/data/SetPgUserauth.sh

# 刪除既有的 haproxy.cfg ，然後複製模板產生新的
rm -f "$FILE_HAPROXY_CONF"
cat ./RedisACLCluster/haproxy/conf/haproxy.cfg.template > "$FILE_HAPROXY_CONF"

# 建立 ACL Redis Cluster
# 建立 redis nodes
docker run -itd \
    --network RedisACLNet \
    --name redis_acl_m \
    --env REDIS_REP_PASS=$REDIS_REP_PASS \
    -v ./RedisACLCluster/m/conf:/usr/local/etc/redis \
    -v ./RedisACLCluster/m:/data \
    -v "$FILE_REDIS_ACL":/etc/redis/users.acl \
    redis:latest \
    ./SetRedisConfFile.sh

for i in $(seq 1 $REPLICA_NUM);
do
    runuser -u jwtauth -- cp -R ./RedisACLCluster/s_template ./RedisACLCluster/s$i
    docker run -itd \
        --network RedisACLNet \
        --name redis_acl_s$i \
        --env REDIS_REP_PASS=$REDIS_REP_PASS \
        -v ./RedisACLCluster/s$i/conf:/usr/local/etc/redis \
        -v ./RedisACLCluster/s$i:/data \
        -v "$FILE_REDIS_ACL":/etc/redis/users.acl \
        redis:latest \
        ./SetRedisConfFile.sh
    sed -e '/#InsertReplicasAbove/i \\tserver redis_acl_s'"$i"' redis_acl_s'"$i"':6379 maxconn 1024 check inter 1s' -i "$FILE_HAPROXY_CONF"
done

sed -e 's/#InsertReplicasAbove//g' -i "$FILE_HAPROXY_CONF"

echo "Replica nodes are up..."
    
# 取得 redis_acl_m 的 IP address
MASTER_REDIS_NODE_IP=$(docker inspect --format='{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' redis_acl_m)

for i in $(seq 1 $SENTINEL_NUM);
do
    runuser -u jwtauth -- cp -R ./RedisACLCluster/st_template ./RedisACLCluster/st$i

    # 建立 sentinels    
    docker run -itd \
        --network RedisACLNet \
        --name redis_acl_st$i \
        --env MASTER_REDIS_NODE_IP=$MASTER_REDIS_NODE_IP \
        --env REDIS_MASTER_AUTH_PASS=$REDIS_MASTER_AUTH_PASS \
        --env QUORUM_NUM=$ST_QUORUM_NUM \
        -v ./RedisACLCluster/st$i/conf:/usr/local/etc/redis \
        -v ./RedisACLCluster/st$i:/data \
        redis:latest \
        ./SetSentinelConfFile.sh
done

echo "Sentinel nodes are up..."

# 再等容器建好
sleep 10s

# 建立 HAProxy
docker run -itd \
    --network RedisACLNet \
    --name RedisACLHAProxy \
    -p 20010:20010 \
    -v ./RedisACLCluster/haproxy/conf:/usr/local/etc/haproxy:ro \
    --env REDISOBSERVER=haproxy \
    --env REDISPASS=$HAPROXY_AUTH_PASS \
    --sysctl net.ipv4.ip_unprivileged_port_start=0 \
    haproxy:latest bash
    
docker exec -d -u haproxy RedisACLHAProxy haproxy -f /usr/local/etc/haproxy/haproxy.cfg

echo "HAProxy node are up..."    

# 建立 JwtAuthSvr 容器
docker run -itd --network JwtNet --name JwtAuthSvr --env SYS_TZONE=$SYS_TZONE -p $JWT_AUTH_PORT:8080 -v ./JWTAuthSvr:/app -w /app alpine:latest

echo "JwtAuthSvr node are up..." 

# 修改 JwtAuthSvr 容器時區
docker exec -it JwtAuthSvr ./SetTZone.sh

# 產生測試憑證
sed -e 's/{IP}/'"$JWT_AUTH_IP_OR_FQDN"'/g' -i ./JWTAuthSvr/SSL/ssl.conf
openssl req -x509 -new -nodes -sha256 -utf8 -days 3650 -newkey rsa:2048 \
	-keyout ./JWTAuthSvr/SSL/ForTest.key \
	-out ./JWTAuthSvr/SSL/ForTest.crt \
	-extensions v3_req \
	-config ./JWTAuthSvr/SSL/ssl.conf

# 處理 alpine 容器信任自簽憑證
docker exec -d JwtAuthSvr sh -c "cat /app/SSL/ForTest.crt >> /etc/ssl/certs/ca-certificates.crt"

# 再等容器建好
sleep 5s

# 打通 JWTAuth 容器到 RedisACLNet 網路的通道
docker network connect RedisACLNet JwtAuthSvr

# 執行 JWTAuth 主程式
docker exec -d JwtAuthSvr ./JWTAuth
