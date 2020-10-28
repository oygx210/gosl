#!/bin/bash

set -e

unamestr=`uname`
if [[ "$unamestr" != 'Linux' ]]; then
   echo "You do not need to call this script on non-Linux systems"
   exit 1
fi

CFLAGS="`pkg-config --silence-errors --cflags hdf5 || true` -D_LARGEFILE64_SOURCE -D_LARGEFILE_SOURCE -D_FORTIFY_SOURCE=2 -g -O2 -Wformat -Werror=format-security"
LDFLAGS="`pkg-config --silence-errors --libs-only-L hdf5 || true` -lhdf5_hl -lhdf5  -lpthread -lz -ldl -lm"

FLAGS_FILE="xautogencgoflags.go"

echo "// Copyright 2019 The Gosl Authors. All rights reserved." > $FLAGS_FILE
echo "// Use of this source code is governed by a BSD-style" >> $FLAGS_FILE
echo "// license that can be found in the LICENSE file." >> $FLAGS_FILE
echo "" >> $FLAGS_FILE
echo "// *** NOTE: this file was auto generated by all.bash ***" >> $FLAGS_FILE
echo "// ***       and should be ignored                    ***" >> $FLAGS_FILE
echo "" >> $FLAGS_FILE
echo "package h5" >> $FLAGS_FILE
echo "" >> $FLAGS_FILE
echo "/*" >> $FLAGS_FILE
echo "#cgo CFLAGS: $CFLAGS" >> $FLAGS_FILE
echo "#cgo LDFLAGS: $LDFLAGS" >> $FLAGS_FILE
echo "*/" >> $FLAGS_FILE
echo "import \"C\"" >> $FLAGS_FILE