# JWTAuth [![doc](https://img.shields.io/badge/doc-80%-brightgreen)]

## 基本介紹

* 這是一個 JWT based authentication server 的 implementation
* 注意，這個服務只提供確認使用者身份和來源的機器，而且最好在同一個網路中執行。如果要跨 proxy 或是 NAT 等等環境，要確認是否支持 X-Forwarded-For (XFF) header
* 使用者使用 Bearer token 從申請 JWT 的同一台機器發送 request 就可以通過認證
* 把使用者認證跟授權的服務分開。回傳的認證資料中會帶有這個 token 的使用者 ID ，可以據此自己實做授權服務
* 目前使用 apache jmeter 實測，每秒鐘可以承受 1000 個 requests  
    (硬體配置 Intel(R) Xeon(R) CPU E5-4610 v2 @ 2.30GHz 8 cores + 16 GB RAM + 100GB storage + 10GbE)
* 完全使用容器架構，並且只使用 dockerhub 上 official 的 image 建構系統
* 使用 redis cluster + HAProxy 來實做一讀多寫的 HA 架構。以此為基礎建構 session server
* 套用 redis 的 acl 機制做 redis 權限控管
* 使用 PostgreSQL 儲存使用者的資料
* 使用 alpine linux 編譯 JWTAuth 主程式
* 設定好環境變數以後 sudo 執行 SetJWTAuth&#46;sh 就可以自動建構整個系統


## 特別注意
* 使用者管理員的密碼預設是 #JWTAuth1234# ，請自己修改。有兩種方式：
1. 自己連上 PostgreSQL server (server domain name:25432) ，然後自己下 update 指令更新 UserMgr 的 密碼 hash ，計算 hash 的方式是
```
這裡要注意 pwhash 的產生方法。
1). 先找到 JWTAuth 應用程式使用的 .env.template ，打開它。
2). 找到 USER_PASS_SALT ，這就是要加在密碼後面的 salt 。
3). pwhash = sha256(密碼 + USER_PASS_SALT)
4). 預設密碼 #JWTAuth1234# ，請記得一定要修改。
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

## 基本需求

* 安裝 docker 服務
* 安裝 OpenSSL
* 要有網路可以接 dockerhub

## 安裝指令

* 建立 OS 系統的 jwtauth 帳號
    ```sudo useradd -m jwtauth```

* 給予 JWTAuth 帳號可以操作 docker 的權限
    ```sudo usermod -aG docker jwtauth```

* 使用 jwtauth 身分執行以下指令
    ```su jwtauth```

* 修改 ./JWTAuthSys/SetJWTAuth.sh ，把裡面的環境變數密碼區的設定改一下
    ```
    # 設立四個 postgresql 密碼環境變數`
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

    # 設定 JWTAuth 安全參數（沒設後果自負）
    # JWT_SEC_KEY 是驗證 JWT 有效性的 key ，一定要自己重設
    # 使用者密碼會使用 USER_PASS_SALT 做 salted hash ，請自己重設
    # IP 一定要記得改
    JWT_SEC_KEY="696ceb369e628963ddd6e17ba4acc76c9a812d19fbfaad68d58581ca513e76e0"
    USER_PASS_SALT="ba541f1d5d01df17b01833f3255b722d540acd719bedc05af8091ac9d40e1f8e"  
    JWT_AUTH_IP="xx.xx.xx.xx"  
    JWT_AUTH_PORT="20001"
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

* 執行 SetJWTAuth&#46;sh
    ```
    sudo ./JWTAuthSys/SetJWTAuth.sh
    ```
