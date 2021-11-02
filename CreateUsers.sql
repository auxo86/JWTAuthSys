-- 建立資料庫
create database userauth;
-- 把要建立的 role 和使用者都用 superuser 建好。
create role userauthadmin;
create role userauthop;
create role userauthqry;
create user jwtauthadmin with password :'PassAdmin';
create user jwtauthop with password :'PassOp';
create user jwtauthqry with password :'PassQry';
-- 授權給管理者
grant all on database userauth to userauthadmin;
-- grant role 一定要用 superuser
grant userauthadmin to jwtauthadmin;
grant userauthop to jwtauthop;
grant userauthqry to jwtauthqry;
