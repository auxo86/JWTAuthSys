# JWTAuth ![doc](https://img.shields.io/badge/doc-80%25-brightgreen)

## 基本介紹

* 這是一個 JWT based authentication server 的 implementation
* 注意，這個服務只提供確認使用者身份和來源的機器，而且最好在同一個網路中執行。如果要跨 proxy 或是 NAT 等等環境，要確認是否支持 X-Forwarded-For (XFF) header
* 使用者使用 Bearer token 從申請 JWT 的同一台機器發送 request 就可以通過認證
* 把使用者認證跟授權的服務分開。回傳的認證資料中會帶有這個 token 的使用者 ID ，可以據此自己實做授權服務
* 目前使用 [apache jmeter](https://jmeter.apache.org/) 實測，每秒鐘可以承受 20000 個 requests  
    (硬體配置 Intel(R) Xeon(R) CPU E5-4610 v2 @ 2.30GHz 8 cores + 16 GB RAM + 100GB storage + 10GbE)
* 完全使用容器架構，並且只使用 dockerhub 上 official 的 image 建構系統
* 使用 redis cluster + HAProxy 來實做一寫多讀的 HA 架構。以此為基礎建構 session server
* 套用 redis 的 acl 機制做 redis 權限控管
* 使用 PostgreSQL 儲存使用者的資料
* 使用 alpine linux 編譯 JWTAuth 主程式
* 設定好環境變數以後 sudo 執行 SetJWTAuth&#46;sh 就可以自動建構整個系統

## 使用情境

現今在微服務的環境下，webapi 扮演著重要的角色。但是如何限制存取就變成了一個重要的問題。JWTAuth 實做了 JWT ，在內容中埋入了申請 JWT 時裝置的 IP 。當使用者 login 的時候會回傳 JWT ，同時會在 redis cluster 中產生一筆 session 紀錄。當 JWT 過期或是 session 過期，驗證會失效。當 JWT 中的 IP 跟使用 JWT 的裝置 IP 不一致的時候驗證也會失效。意思就是這個 JWT 不能跨裝置使用。

假設有個 webapi A 提供服務，在 client 呼叫 A 時， client 應該在 http request 的 header 中攜帶申請來的 JWT ，當 A 收到呼叫以後，取出這個 JWT ，同時取得 http request 的 IP address ，然後重新建立封包，把 JWT 和 IP address (以 JSON 的形式) 傳送給 JWTAuth server ， JWTAuth 服務在驗證 JWT 和 ip address 過後會回傳使用者的 UserID ，於是收到回應的 A 可以根據這個 UserID 自己實做相對應的權限系統，決定應該賦予 client 什麼權限。

## 特別注意

使用者管理員的密碼預設是 #JWTAuth1234# ，請自己修改。有兩種方式：

1. 自己連上 PostgreSQL server (server domain name:25432) ，然後自己下 update 指令更新 UserMgr 的 密碼 hash ，計算 hash 的方式是

```
這裡要注意 hash 的產生方法。
1). 使用 "golang.org/x/crypto/bcrypt"
2). // 先設定密碼
    password := []byte("#JWTAuth1234#")
    // 生成隨機 salt 值並使用默認的 cost 值進行 hash
    hash, err := bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
3). 預設密碼 #JWTAuth1234# ，請記得一定要修改。
```

2. 使用 webapi 更新帳號 UserMgr 的密碼。

## webapi 介紹

* user  

__/login__: 使用者登入，傳入帳號密碼以後可以回傳 JWT ，如果重複的 login ，會回傳不同的 JWT ，但是 http status 會變成 201  

__/JwtValidation__: JWT 驗證，如果通過驗證會回傳使用者的 UserID  

__/logout__: 使用者登出，會刪除 session server 上的 session 。  

* 使用者管理員  

__/AddOneUser__: 供 UserMgr 新增"一位"使用者  

__/BatchAddUsers__: 供 UserMgr 一次新增"一批"使用者  

__/GetUserData__: 供 UserMgr 向 userauth database 查詢一個使用者  

__/UpdOneUser__: 供 UserMgr 向 userauth database 更新一個新使用者。如果要更新使用者密碼也使用這個 webapi  

__/DeleteOneSession__: 供 UserMgr 刪除 redis server 上的 session  

為了讓大家容易測試，特別附上了 jmeter 使用的[測試檔](https://github.com/auxo86/JWTAuthSys/blob/main/JWTAuthTest.jmx)。請特別注意，使用 __/login__ 取得 JWT 後，請放入各個測試 webapi 的 Bearer token 的位置。記得修改 webapi 的 ip 或是 FQDN ，否則會無法測試。  

請注意這個測試檔案中的各個 JSON 的範例，這個就是要呼叫 webapi 時 http request body 中要填入的 JSON 內容。  

當然不想用 jmeter 也是可以用 cURL 來測試。例如，如果我要 call __/AddOneUser__ ，可以這樣做：  

```
curl -k -d '{ "iUserCatID":1, "sUserID":"TonyStark", "sUserName":"東尼·史塔克", "sPassword":"!TonyStark!" }' \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${JWT}" \
    -X POST https://xxx.xxx.xxx.xxx:20001/AddOneUser
```

## 基本需求

* 安裝 docker 服務
* 安裝 OpenSSL
* 要有網路可以接 dockerhub

## 安裝指令

* 建立 OS 系統的 jwtauth 帳號

    ```
    sudo useradd -m jwtauth
    ```

* 給予 jwtauth 帳號可以操作 docker 的權限

    ```
    sudo usermod -aG docker jwtauth
    ```

* 給予 jwtauth 帳號可以操作 sudo 的權限

    ```
    sudo usermod -aG sudo jwtauth
    ```

* 使用 jwtauth 身分執行以下指令

    ```
    su jwtauth && cd ~
    ```

* 修改 ./JWTAuthSys/SetJWTAuth.sh ，把裡面的環境變數區的設定改一下

    ```
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

    # 設定 sentinel nodes 的數目
    SENTINEL_NUM=5
    ```

    __20001__ 是預設值，請依照 SetJWTAuth&#46;sh 中建立 JWTAuthSvr 容器開放的 port 決定這個值。

* 修改 ./JWTAuthSys/JWTAuthSvr/.env.template

    ```
    # 憑證設定
    sshCert=./SSL/ForTest.crt
    sshKey=./SSL/ForTest.key
    ```

    雖然在這裡會暫時提供自簽的憑證，  
    但是如果你使用自簽的憑證，就要自己處理憑證信任問題。  
    這一點請特別注意。  
    設定檔請依實際條件修改。  
    
    如果使用的是 __合法的憑證__ ，請在 &#46;/JWTAuthSys/SetJWTAuth&#46;sh 中註解以下幾行。  
    
    ```sh
    # 產生測試憑證
    sed -e 's/{IP}/'"$JWT_AUTH_IP_OR_FQDN"'/g' -i /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ssl.conf
    openssl req -x509 -new -nodes -sha256 -utf8 -days 3650 -newkey rsa:2048 \
        -keyout /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ForTest.key \
        -out /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ForTest.crt \
        -extensions v3_req \
        -config /home/jwtauth/JWTAuthSys/JWTAuthSvr/SSL/ssl.conf

    # 處理 alpine 容器信任自簽憑證
    docker exec -d JwtAuthSvr sh -c "cat /app/SSL/ForTest.crt >> /etc/ssl/certs/ca-certificates.crt"
    ```

* 執行 SetJWTAuth&#46;sh

    ```
    sudo ./JWTAuthSys/SetJWTAuth.sh
    ```
