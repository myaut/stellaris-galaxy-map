#!/bin/bash

DESTDIR=$(dirname $0)/names
STELLARISDIR="$1"
if ! [ -d "$1" ]; then 
	echo "Usage: import_names.sh STELLARISDIR"
	exit 1
fi

LOCDIR=$STELLARISDIR/localisation/english

mkdir -p names

grep -v "_desc:" $LOCDIR/prescripted_l_english.yml > $DESTDIR/prescripted.yml

cp $LOCDIR/empire_formats_l_english.yml $DESTDIR/empire_formats.yml

cat $LOCDIR/species_*_l_english.yml > $DESTDIR/species.yml

egrep 'species_name' $LOCDIR/prescripted_countries_names_l_english.yml > $DESTDIR/prescripted_countries.yml
