#!/bin/bash
sed -e "s/{MasterAuthPass}/$REDIS_MASTER_AUTH_PASS/g" < /usr/local/etc/redis/sentinel.conf > /etc/sentinel.conf && redis-server /etc/sentinel.conf --sentinel
