#!/bin/bash
sed -e "s/{RepPass}/$REDIS_REP_PASS/g" < /usr/local/etc/redis/redis.conf > /etc/redis.conf && redis-server /etc/redis.conf --appendonly yes
