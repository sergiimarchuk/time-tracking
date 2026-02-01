#!/bin/bash
cd /opt/web-trckr-1
/usr/bin/docker-compose down
echo "$(date): Worklog tracker stopped" >> /var/log/worklog-schedule.log
