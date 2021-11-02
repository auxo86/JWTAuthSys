#!/bin/bash

# 建立使用者群，注意，這裡要"容器內"絕對路徑
psql -U postgres --set=PassAdmin="$PASS_ADMIN" --set=PassOp="$PASS_OP" --set=PassQry="$PASS_QRY" -f /var/lib/postgresql/data/CreateUsers.sql
# grant 預設的物件權限
psql -U jwtauthadmin -d userauth -f /var/lib/postgresql/data/GrantPrivileges.sql
# 在 PgUserAuth 中塞入預先定義好的 schema
psql -U jwtauthadmin -d userauth -f /var/lib/postgresql/data/CreateUserAuthDDL.sql
