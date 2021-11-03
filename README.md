# JWTAuth 安裝說明

## 基本需求

* 安裝 docker 服務
* 安裝 OpenSSL

## 安裝指令

* 建立 OS 系統的 jwtauth 帳號

    ```sudo useradd -m jwtauth```

* 給予 JWTAuth 帳號可以操作 docker 的權限

    ```sudo usermod -aG docker jwtauth```

* 使用 jwtauth 身分執行以下指令

    ```su jwtauth```

* 修改 ./JWTAuthSys/SetJWTAuth.sh ，把裡面的環境變數密碼區的設定改一下

    `# 設立四個 postgresql 密碼環境變數`  
    ```
    PG_SUPER_PASS="#JWTAuth1234#"
    PG_ADMIN_PASS="#JWTAuth1234#"
    PG_OP_PASS="#JWTAuth1234#"
    PG_QRY_PASS="#JWTAuth1234#"
    ```

    `# 設定 redis cluster 中會用到的密碼環境變數`  
    ```
    REDIS_OP_PASS="#JWTAuth1234#"
    REDIS_READER_PASS="#JWTAuth1234#"  
    REDIS_REP_PASS="#JWTAuth1234#"  
    REDIS_MASTER_AUTH_PASS="#JWTAuth1234#"  
    HAPROXY_AUTH_PASS="#JWTAuth1234#"
    ```

    `# 設定 JWTAuth 安全參數`  
    ```
    JWT_SEC_KEY="696ceb369e628963ddd6e17ba4acc76c9a812d19fbfaad68d58581ca513e76e0"
    USER_PASS_SALT="ba541f1d5d01df17b01833f3255b722d540acd719bedc05af8091ac9d40e1f8e"  
    JWT_AUTH_IP="xx.xx.xx.xx"  
    JWT_AUTH_PORT="20001"
    ```
  
    __20001__ 是預設值，請依照 SetJWTAuth&#46;sh 中建立 JWTAuthSvr 開放的 port 決定這個值。  

* 修改 ./JWTAuthSys/JWTAuthSvr/.env.template

    `# 憑證設定`  
    ```
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
