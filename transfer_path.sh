# change import path 
# sofastack.io/sofa-mosn to mosn.io/mosn
REPSTR="s/sofastack.io\/sofa-mosn/mosn.io\/mosn/g"
GREPSTR="grep sofastack.io/sofa-mosn -rl"
CODES=("./pkg/" "./examples/codes/" "./test/" "./cmd/")
# valid for MacOS (Drawin)
# if you run the script in MacOS, you should use this one
#SED_CMD="sed -i ''"
# valid for Linux
# if you run the script in Linux, you should use this one
#SED_CMD="sed -i"

# auto detect system
SYSTEM=`uname -s`

# valid for Linux
OS="Linux"
SED_CMD="sed -i"

# valid for MacOS (Drawin)
if [ $SYSTEM = "Darwin" ] ; then
	OS="Darwin"
	SED_CMD="sed -i ''"
fi

for CODE in "${CODES[@]}"
do
	CMD="$SED_CMD \"$REPSTR\" \`$GREPSTR $CODE\`"
	eval "$CMD"
done