#/bin/bash

DUMP=""
DUMP_LOCATION=$1
if [ -d "$DUMP_LOCATION" ]
then
    DUMP="-d $DUMP_LOCATION"
fi

if [ -f "$DUMP_LOCATION" ]
then
    DUMP="-f $DUMP_LOCATION"
fi

list=""
test_command () {
    rt=$(eval $1 2>/dev/null)
    if [ $? -ne 0 ] 
    then
        echo Failed: $1
        echo $rt
        echo
    else
        echo Passed: $1
    fi
    list=$(eval $1 2>/dev/null| head -n2 | tail -n1)
}

test_res () {
    test_command "go run cmd/main.go $DUMP get $1 -A"
    ns=$(echo $list | cut -f1 -d' ')
    item=$(echo $list | cut -f2 -d' ')
    test_command "go run cmd/main.go $DUMP describe $1 $item -n $ns"
}

test_command "go run cmd/main.go $DUMP show"
test_command "go run cmd/main.go $DUMP get no"
item=$(echo $list | cut -f1 -d' ')
test_command "go run cmd/main.go $DUMP describe no $item"

test_command "go run cmd/main.go $DUMP get pv"
item=$(echo $list | cut -f1 -d' ')
test_command "go run cmd/main.go $DUMP describe pv $item"

test_command "go run cmd/main.go $DUMP get sc"
item=$(echo $list | cut -f1 -d' ')
test_command "go run cmd/main.go $DUMP describe sc $item"

test_command "go run cmd/main.go $DUMP get clusterrole"
item=$(echo $list | cut -f1 -d' ')
test_command "go run cmd/main.go $DUMP describe clusterrole $item"

test_command "go run cmd/main.go $DUMP get clusterrolebinding"
item=$(echo $list | cut -f1 -d' ')
test_command "go run cmd/main.go $DUMP describe clusterrolebinding $item"

test_res "po"
test_res "svc"
test_res "deploy"
test_res "ds"
test_res "rs"
test_res "sts"
test_res "event"
test_res "pvc"
test_res "secret"
test_res "cm"
test_res "sa"
test_res "ing"
test_res "ep"
test_res "job"
test_res "cronjob"
test_res "role"
test_res "rolebinding"