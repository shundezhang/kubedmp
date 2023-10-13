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
    rt=$(eval $1 2>/dev/null | head -n2 | tail -n1)
    if [ $? -ne 0 ] 
    then
        echo Failed: $1
        echo $rt
        echo
    else
        echo Passed: $1
    fi
    list=$rt
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

test_res "po"
test_res "svc"
test_res "deploy"
test_res "ds"
test_res "rs"
test_res "sts"
test_res "event"
test_res "pv"
test_res "pvc"
test_res "secret"
test_res "cm"
test_res "sa"
test_res "ing"
