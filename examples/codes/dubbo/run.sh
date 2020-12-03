#!/usr/bin/env bash


build() {
    pushd ${DUBBO_ROOT_DIR}
    mvn clean package
    popd
}

java_options() {
    JAVA_OPTIONS="-server -Xmx1g -Xms1g -XX:MaxDirectMemorySize=1g -XX:+UseG1GC"
}

run() {
    if [ -d "${PROJECT_DIR}/target" ]; then
        JAR=`find ${PROJECT_DIR}/target/*.jar | head -n 1`
        CMD="java ${JAVA_OPTIONS}  -jar ${JAR}"
        echo "command is: ${CMD}"
        ${CMD}
    fi
}

JAVA_OPTIONS=""
DUBBO_ROOT_DIR="dubbo-examples"
PROJECT_DIR=""
RUN_OPTION=$1

if [ "${RUN_OPTION}" == "server" ]; then
    PROJECT_DIR=${DUBBO_ROOT_DIR}"/dubbo-examples-provider"
elif [ "${RUN_OPTION}" == "client" ]; then
    PROJECT_DIR=${DUBBO_ROOT_DIR}"/dubbo-examples-consumer"
else
    echo "[ERROR] Invalid option ${RUN_OPTION}, usage: sh run.sh server or sh run.sh client"
    exit 0
fi


build
java_options
run






