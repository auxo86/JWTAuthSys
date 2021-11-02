-- 建立 schema
create schema users;
-- 配置預設權限給 role ，這個可以用 database 管理者做 (非 superuser)，注意權限必須有 select ，因為要能 update 必須同時有 select 權限 @@
alter default privileges in schema users grant select, insert, update, delete on tables to userauthop;
alter default privileges in schema users grant select on tables to userauthqry;
-- 除了 tables 以外，有時候會用到遞增數字欄位，也要配置給 sequences 存取權限，否則操作者差入資料會因為無法使用 sequences 而失敗
alter default privileges in schema users grant usage, select on sequences to userauthop;
-- grant 使用 schema 的權限。
grant usage on schema users to userauthop;
grant usage on schema users to userauthqry;
