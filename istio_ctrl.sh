#!/bin/bash


# There are dependency conflicts between different istio versions.
# the go mod will ignore dirctories that are start with "_", so we rename the others versions to it.
target=$1

for i in $(ls ./istio | grep _); do foo=${i#"_"};mv istio/$i istio/$foo; done; # clean all ignored dirctories
for i in $(ls ./istio | grep -v $target); do mv istio/$i istio/_$i;done; # move others to ignored. (start with _)
