#!/usr/bin/env bash
#
# build docker image
#

build()
{
    echo -e "building image: log-pilot:latest\n"

    docker build -t log-pilot:latest -f Dockerfile.$1 .
    docker tag log-pilot:latest edj-docker-registry-vpc.cn-hangzhou.cr.aliyuncs.com/edj-public/edj-log-pilot:2.0
    docker push edj-docker-registry-vpc.cn-hangzhou.cr.aliyuncs.com/edj-public/edj-log-pilot:2.0
}

case $1 in
fluentd)
    build fluentd
    ;;
*)
    build filebeat
    ;;
esac
