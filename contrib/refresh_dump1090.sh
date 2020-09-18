#/usr/bin/env bash
version=$1
if [ "$version" = "" ]; then
    echo "missing dump1090 version!"
    exit 1
fi
tmpdir=$(mktemp -d)
gitroot=$(git rev-parse --show-toplevel)
cd $tmpdir
git clone https://github.com/flightaware/dump1090
cd dump1090
git checkout "$version"
rm -rf $gitroot/resources/dump1090
mkdir -p $gitroot/resources/dump1090/
cp -a $tmpdir/dump1090/public_html $gitroot/resources/dump1090/public_html
rm -rf $tmpdir/dump1090
rmdir $tmpdir
