#!/bin/bash

sync_dir=~/Downloads/movements
copy_to_dir=~/go/gofit/internal/static/webpage/movement_images

[ -d "$sync_dir" ] || exit 1
[ -d "$copy_to_dir" ] || exit 1

for i in $sync_dir/*; do
	p=`basename $i`
	p=`echo "$p" | awk '{print tolower($0)}'`
	p=`echo $p | sed 's/_/ /g'`
	filename=`echo $p | awk '{print $NF}'`
	directory=`echo $p | sed "s/ $filename$//"`
	destdir="$copy_to_dir/$directory"
	[ -d "$destdir" ] || mkdir "$destdir"
	dest="$destdir/$filename"
	echo "Syncing $i to $dest"
	cp "$i" "$dest"
done

