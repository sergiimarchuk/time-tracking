#!/bin/bash
cd /opt/web-trckr-1
/usr/bin/docker-compose up -d
echo "$(date): Worklog tracker started" >> /var/log/worklog-schedule.log
