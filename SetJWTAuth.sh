#!/bin/bash

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

# 設定 redis acl file 的密碼
sed -e 's/{RedisOpPass}/'"$REDIS_OP_PASS"'/g' -i /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template
sed -e 's/{RedisReaderPass}/'"$REDIS_READER_PASS"'/g' -i /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template
sed -e 's/{RedisRepPass}/'"$REDIS_REP_PASS"'/g' -i /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template
sed -e 's/{RedisSentPass}/'"$REDIS_MASTER_AUTH_PASS"'/g' -i /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template
sed -e 's/{HaproxyPass}/'"$HAPROXY_AUTH_PASS"'/g' -i /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template

# 產生 redis 使用的 users.acl
cat /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl.template > /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl
chmod 400 /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl

# 設定 JWTAuth 安全參數
JWT_SEC_KEY="696ceb369e628963ddd6e17ba4acc76c9a812d19fbfaad68d58581ca513e76e0"
USER_PASS_SALT="ba541f1d5d01df17b01833f3255b722d540acd719bedc05af8091ac9d40e1f8e"
JWT_AUTH_IP="1.2.3.4"
JWT_AUTH_PORT="20001"

# 設定 JWTAuth .env 秘密參數
sed -e 's/{RedisOpPass}/'"$REDIS_OP_PASS"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{RedisReaderPass}/'"$REDIS_READER_PASS"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{PgQryPass}/'"$PG_QRY_PASS"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{PgOpPass}/'"$PG_OP_PASS"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{JwtSecKey}/'"$JWT_SEC_KEY"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{PassSalt}/'"$USER_PASS_SALT"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{AuthSvrIp}/'"$JWT_AUTH_IP"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template
sed -e 's/{AuthSvrPort}/'"$JWT_AUTH_PORT"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template

# 產生 .env 設定檔
cat /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env.template > /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env
chmod 400 /home/jwtauth/JWTAuthSys/JWTAuthSvr/.env

# 如果是外網，可以使用 docker pull 取得官方的 images ，然後需要對 alpine linux 做 WORKDIR 的設定，然後重新 commit 成 alpine_env:latest 。
docker pull postgres
docker pull redis
docker pull haproxy
docker pull alpine

# 抓取 GO 編譯環境
docker pull golang:alpine

# 編譯 JWTAuth
docker run --rm -v /home/jwtauth/JWTAuthSys/src:/usr/src -w /usr/src/JWTAuth golang:alpine go build -v
chown jwtauth:jwtauth /home/jwtauth/JWTAuthSys/src/JWTAuth/JWTAuth
chmod 500 /home/jwtauth/JWTAuthSys/src/JWTAuth/JWTAuth
mv /home/jwtauth/JWTAuthSys/src/JWTAuth/JWTAuth /home/jwtauth/JWTAuthSys/JWTAuthSvr/

# 建立 GO 執行環境的映像檔
# 修改 alpine image ，增加 WORKDIR 然後重新 build 成新的 image alpine_env:latest
# mkdir ./tmp && echo -e "FROM alpine\nRUN mkdir /app\nWORKDIR /app" | docker build -t alpine_env:latest -f- ./tmp && rm -rf ./tmp && docker rmi alpine:latest

# 建立網路
docker network create JwtNet
docker network create RedisACLNet

# 建立資料夾給 postgresql 資料庫容器 mount 使用。
mkdir /home/jwtauth/JWTAuthSys/ForPgUserAuth
chown jwtauth:jwtauth /home/jwtauth/JWTAuthSys/ForPgUserAuth

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
    -v /home/jwtauth/JWTAuthSys/ForPgUserAuth:/var/lib/postgresql/data \
    postgres:latest
	
# 等容器建好
sleep 10s

# 修正資料庫時區
sed -e 's/Etc\/UTC/Asia\/Taipei/g' -i /home/jwtauth/JWTAuthSys/ForPgUserAuth/postgresql.conf

# 把要執行的 SQL 複製到容器 binding 的目錄下
cp /home/jwtauth/JWTAuthSys/*.sql /home/jwtauth/JWTAuthSys/ForPgUserAuth
cp /home/jwtauth/JWTAuthSys/SetPgUserauth.sh /home/jwtauth/JWTAuthSys/ForPgUserAuth

# 重載系統設定
docker exec -d -u postgres PgUserAuth pg_ctl reload

# 建立使用者群、grant 預設的物件權限、在 PgUserAuth 中塞入預先定義好的 schema 。注意，這裡要"容器內"絕對路徑。
docker exec -it PgUserAuth /var/lib/postgresql/data/SetPgUserauth.sh

# 建立 ACL Redis Cluster
# 建立 redis nodes
docker run -itd \
    --network RedisACLNet \
    --name redis_acl_m \
    --env REDIS_REP_PASS=$REDIS_REP_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/m/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/m:/data \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl:/etc/redis/users.acl \
    redis:latest \
    ./SetRedisConfFile.sh

docker run -itd \
    --network RedisACLNet \
    --name redis_acl_s1 \
    --env REDIS_REP_PASS=$REDIS_REP_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/s1/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/s1:/data \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl:/etc/redis/users.acl \
    redis:latest \
    ./SetRedisConfFile.sh
	
docker run -itd \
    --network RedisACLNet \
    --name redis_acl_s2 \
    --env REDIS_REP_PASS=$REDIS_REP_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/s2/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/s2:/data \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/users.acl:/etc/redis/users.acl \
    redis:latest \
    ./SetRedisConfFile.sh

# 建立 sentinels    
docker run -itd \
    --network RedisACLNet \
    --name redis_acl_st1 \
    --env REDIS_MASTER_AUTH_PASS=$REDIS_MASTER_AUTH_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st1/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st1:/data \
    redis:latest \
    ./SetSentinelConfFile.sh

docker run -itd \
    --network RedisACLNet \
    --name redis_acl_st2 \
    --env REDIS_MASTER_AUTH_PASS=$REDIS_MASTER_AUTH_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st2/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st2:/data \
    redis:latest \
    ./SetSentinelConfFile.sh
	
docker run -itd \
    --network RedisACLNet \
    --name redis_acl_st3 \
    --env REDIS_MASTER_AUTH_PASS=$REDIS_MASTER_AUTH_PASS \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st3/conf:/usr/local/etc/redis \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/st3:/data \
    redis:latest \
    ./SetSentinelConfFile.sh
	
# 再等容器建好
sleep 10s

# 建立 HAProxy
docker run -itd \
    --network RedisACLNet \
    --name RedisACLHAProxy \
    -p 20010:20010 \
    -v /home/jwtauth/JWTAuthSys/RedisACLCluster/haproxy/conf:/usr/local/etc/haproxy:ro \
    --env REDISOBSERVER=haproxy \
    --env REDISPASS=$HAPROXY_AUTH_PASS \
    --sysctl net.ipv4.ip_unprivileged_port_start=0 \
    haproxy:latest

# 建立 JWTAuthSvr 容器
docker run -itd --network JwtNet --name JwtAuthSvr -p 20001:8080 -v /home/jwtauth/JWTAuthSys/JWTAuthSvr:/app -w /app alpine:latest

# 產生測試憑證
sed -e 's/{IP}/'"$JWT_AUTH_IP"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ssl.conf
openssl req -x509 -new -nodes -sha256 -utf8 -days 3650 -newkey rsa:2048 \
	-keyout /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ForTest.key \
	-out /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ForTest.crt \
	-extensions v3_req \
	-config /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ssl.conf

# 處理 alpine_env 容器信任自簽憑證
docker exec -d JwtAuthSvr cat ./SSL/ForTest.crt >> /etc/ssl/certs/ca-certificates.crt

# 再等容器建好
sleep 5s

# 打通 JWTAuth 容器到 RedisACLNet 網路的通道
docker network connect RedisACLNet JwtAuthSvr

# 執行 JWTAuth 主程式
docker exec -d JwtAuthSvr ./JWTAuth
