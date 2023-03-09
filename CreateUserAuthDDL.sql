-- -- 建立所有表格的基本模板
-- CREATE TABLE TabName
-- (
--     id             serial             NOT NULL, -- 表格列的唯一編號，不一定是 PK
--     cancelflag     smallint DEFAULT 0 NOT NULL,
--     createopid     varchar(30)         NOT NULL, -- 可能是 int，看表格定義
--     modifyopid     varchar(30)         NOT NULL, -- 可能是 int，看表格定義
--     createdatetime timestamptz          NOT NULL,
--     modifydatetime timestamptz          NOT NULL,
-- );

-- -- 插入使用者資料的模板
-- insert into users.usersecret(categoryid, userid, username, pwhash, cancelflag, createopid, modifyopid,
--                              createdatetime,
--                              modifydatetime)
-- values (0, '', '', '', 0, '',
--         '', now(), now());

drop table if exists users.usersecret cascade;
drop table if exists users.usercategory cascade;
drop table if exists users.usersecret_audit cascade;

-- 建立使用者分類表
CREATE TABLE users.usercategory
(
    id             serial              NOT NULL,
    categoryname   varchar(256) UNIQUE NOT NULL,
    cancelflag     smallint DEFAULT 0  NOT NULL,
    createopid     varchar(30)         NOT NULL,
    modifyopid     varchar(30)         NOT NULL,
    createdatetime timestamptz           NOT NULL,
    modifydatetime timestamptz           NOT NULL,
    CONSTRAINT usercategory_pk PRIMARY KEY (id)
);
CREATE INDEX usercategory_idx1 ON users.usercategory USING btree (categoryname varchar_pattern_ops);

-- 建立使用者密碼表
CREATE TABLE users.usersecret
(
    id             serial             NOT NULL,
    categoryid     int                NOT NULL,
    userid         varchar(30)        NOT NULL,
    username       varchar(256)       NOT NULL,
    pwhash         varchar(64)        NOT NULL,
    cancelflag     smallint DEFAULT 0 NOT NULL,
    createopid     varchar(30)         NOT NULL,
    modifyopid     varchar(30)         NOT NULL,
    createdatetime timestamptz          NOT NULL,
    modifydatetime timestamptz          NOT NULL,
    CONSTRAINT usersecret_pk PRIMARY KEY (userid)
);
CREATE INDEX usersecret_idx1 ON users.usersecret USING btree (username varchar_pattern_ops);
ALTER TABLE users.usersecret
    ADD CONSTRAINT "usersecret_fk1" FOREIGN KEY (categoryid) REFERENCES users.usercategory (id);

-- 建立 users.usersecret_audit 表
CREATE TABLE users.usersecret_audit
(
    id             serial             NOT NULL,
    evt_ts         timestamptz,
    session_usr    name,
    remote_address inet,
    remote_port    int,
    evt_op_type    char(1)            NOT NULL,
    id_orig        int                NOT NULL,
    categoryid     int                NOT NULL,
    userid         varchar(30)        NOT NULL,
    username       varchar(256)       NOT NULL,
    pwhash         varchar(64)        NOT NULL,
    cancelflag     smallint DEFAULT 0 NOT NULL,
    createopid     varchar(30)         NOT NULL,
    modifyopid     varchar(30)         NOT NULL,
    createdatetime timestamptz          NOT NULL,
    modifydatetime timestamptz          NOT NULL,
    CONSTRAINT usersecret_audit_pk PRIMARY KEY (id)
);
CREATE INDEX usersecret_audit_idx1 ON users.usersecret_audit USING btree (userid varchar_pattern_ops);
CREATE INDEX usersecret_audit_idx2 ON users.usersecret_audit USING btree (username varchar_pattern_ops);

-- 插入預設使用者群組資料 (第 -1 和第 0 筆的 id 要自己指定。)
INSERT INTO users.usercategory(id, categoryname, cancelflag, createopid, modifyopid, createdatetime, modifydatetime)
	VALUES (-1, 'UserManager', 0, 'UserMgr', 'UserMgr', now(), now());
insert INTO users.usercategory(id, categoryname, cancelflag, createopid, modifyopid, createdatetime, modifydatetime)
	VALUES (0, 'APIUsers', 0, 'UserMgr', 'UserMgr', now(), now());
insert INTO users.usercategory (categoryname, cancelflag, createopid, modifyopid, createdatetime, modifydatetime)
	VALUES ('HumanUsers', 0, 'UserMgr', 'UserMgr', now(), now());

-- 寫 trigger 追蹤表格的變化
CREATE OR REPLACE FUNCTION users.proc_usersecret_audit()
    RETURNS trigger AS
$usersecret_audit$
BEGIN
    IF (tg_op) = 'DELETE' THEN
        INSERT INTO users.usersecret_audit (evt_ts,
                                            session_usr,
                                            remote_address,
                                            remote_port,
                                            evt_op_type,
                                            id_orig,
                                            categoryid,
                                            userid,
                                            username,
                                            pwhash,
                                            cancelflag,
                                            createopid,
                                            modifyopid,
                                            createdatetime,
                                            modifydatetime)
        VALUES (now(),
                session_user,
                inet_client_addr(),
                inet_client_port(),
                'D',
                old.id,
                old.categoryid,
                old.userid,
                old.username,
                old.pwhash,
                old.cancelflag,
                old.createopid,
                old.modifyopid,
                old.createdatetime,
                old.modifydatetime);
        RETURN old;
    ELSIF (TG_OP = 'UPDATE') THEN
        INSERT INTO users.usersecret_audit (evt_ts,
                                            session_usr,
                                            remote_address,
                                            remote_port,
                                            evt_op_type,
                                            id_orig,
                                            categoryid,
                                            userid,
                                            username,
                                            pwhash,
                                            cancelflag,
                                            createopid,
                                            modifyopid,
                                            createdatetime,
                                            modifydatetime)
        VALUES (now(),
                session_user,
                inet_client_addr(),
                inet_client_port(),
                'U',
                new.id,
                new.categoryid,
                new.userid,
                new.username,
                new.pwhash,
                new.cancelflag,
                new.createopid,
                new.modifyopid,
                new.createdatetime,
                new.modifydatetime);
        return new;
    ELSIF (TG_OP = 'INSERT') THEN
        INSERT INTO users.usersecret_audit (evt_ts,
                                            session_usr,
                                            remote_address,
                                            remote_port,
                                            evt_op_type,
                                            id_orig,
                                            categoryid,
                                            userid,
                                            username,
                                            pwhash,
                                            cancelflag,
                                            createopid,
                                            modifyopid,
                                            createdatetime,
                                            modifydatetime)
        VALUES (now(),
                session_user,
                inet_client_addr(),
                inet_client_port(),
                'I',
                new.id,
                new.categoryid,
                new.userid,
                new.username,
                new.pwhash,
                new.cancelflag,
                new.createopid,
                new.modifyopid,
                new.createdatetime,
                new.modifydatetime);
        return new;
    END IF;
    RETURN NULL;
END;
$usersecret_audit$ LANGUAGE plpgsql;

CREATE TRIGGER usersecret_audit
    AFTER INSERT OR UPDATE OR DELETE
    ON users.usersecret
    FOR EACH ROW
EXECUTE PROCEDURE users.proc_usersecret_audit();

-- 插入天字第一號使用者資料 (第一欄的 id 要自己指定。)
-- 這裡要注意 pwhash 的產生方法。
-- 1. 先找到 JWTAuth 應用程式使用的 .env.template ，打開它。
-- 2. 找到 USER_PASS_SALT ，這就是要加在密碼後面的 salt 。
-- 3. pwhash = sha256(密碼 + USER_PASS_SALT)
-- 4. 預設密碼 #JWTAuth1234# ，請記得一定要修改。
insert into users.usersecret(id, categoryid, userid, username, pwhash, cancelflag, createopid, modifyopid,
                             createdatetime,
                             modifydatetime)
values (0, -1, 'UserMgr', '使用者管理員', '$2a$10$PHSBT9M6d32dg550QhdOg.IuH8uOlOafte3peadd8KPF56Kdo1K7y', 0, 'UserMgr', 'UserMgr', now(), now());

-- 這時候才可以建立表格間的 createopid, modifyopid FK
ALTER TABLE users.usercategory
    ADD CONSTRAINT "usercategory_fk1" FOREIGN KEY (createopid) REFERENCES users.usersecret (userid);
ALTER TABLE users.usercategory
    ADD CONSTRAINT "usercategory_fk2" FOREIGN KEY (modifyopid) REFERENCES users.usersecret (userid);
ALTER TABLE users.usersecret
    ADD CONSTRAINT "usersecret_fk2" FOREIGN KEY (createopid) REFERENCES users.usersecret (userid);
ALTER TABLE users.usersecret
    ADD CONSTRAINT "usersecret_fk3" FOREIGN KEY (modifyopid) REFERENCES users.usersecret (userid);
