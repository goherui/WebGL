#!/bin/bash
pkill -f login-server 2>/dev/null
sleep 1
cd /Code/login-page/kratos-login
nohup ./login-server > server.log 2>&1 &
echo "started pid=$!"
sleep 2
cat /Code/login-page/kratos-login/server.log