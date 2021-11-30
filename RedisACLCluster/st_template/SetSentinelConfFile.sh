#!/bin/bash
sed -e "s/{MasterAuthPass}/$REDIS_MASTER_AUTH_PASS/g" < /usr/local/etc/redis/sentinel.conf.template > /etc/sentinel.conf \
&& sed -e 's/{RedisMasterIP}/'"$MASTER_REDIS_NODE_IP"'/g' -i /etc/sentinel.conf \
&& sed -e "s/{Quorum}/$QUORUM_NUM/g" -i /etc/sentinel.conf \
&& redis-server /etc/sentinel.conf --sentinel
