### Prerequisite

+ Install docker on your dev box. Details: https://docs.docker.com/install/
+ Install uinx env if you use windows
 
### Version definition

+ Version pattern: a.b.c
+ a represents the major version, a milestone
+ b for major features in a a milestone
+ c for sub features, minor features, bug fixes, etc.

### Coding

+ coding
+ cli: make test
+ add a term of code change to changelog
+ change version in VERSION file if needed
+ commit code

### Build binary
+ cli: make build
+ get runnable binary in bundles/${version}/binary folder

### Build docker image
+ cli: make image
+ note: commit code before build image

 