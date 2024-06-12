#!/bin/bash

sync_dir=~/Downloads/movements
copy_to_dir=~/go/gofit/internal/server/static/movement_images

[ -d "$sync_dir" ] || exit 1
[ -d "$copy_to_dir" ] || exit 1

for i in $sync_dir/*; do
	p=`basename $i`
	p=`echo "$p" | awk '{print tolower($0)}'`
	p=`echo $p | sed 's/_/ /g'`
	filename=`echo $p | awk '{print $NF}'`
	directory=`echo $p | sed "s/ $filename$//"`
	dest="$copy_to_dir/$directory/$filename"
	echo "Syncing $i to $dest"
	cp "$i" "$dest"
done

