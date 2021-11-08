#!/bin/sh
apk add tzdata
cp /usr/share/zoneinfo/$SYS_TZONE /etc/localtime
echo $SYS_TZONE > /etc/timezone
apk del tzdata
date
echo "Timezone inited. Please check if it is correct."
