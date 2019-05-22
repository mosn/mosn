# change import path 
# github.com/alipay/sofa-mosn to sofastack.io/sofa-mosn
REPSTR="s/github.com\/alipay\/sofa-mosn/sofastack.io\/sofa-mosn/g"
GREPSTR="grep github.com/alipay/sofa-mosn -rl" 
CODES=("./pkg/" "./examples/codes/" "./test/" "./cmd/")
# valid for MacOS (Drawin)
# if you run the script in MacOS, you should use this one
SED_CMD="sed -i ''"
# valid for Linux
# if you run the script in Linux, you should use this one
#SED_CMD="sed -i"

for CODE in "${CODES[@]}"
do
	CMD="$SED_CMD \"$REPSTR\" \`$GREPSTR $CODE\`"
	eval "$CMD"
done
