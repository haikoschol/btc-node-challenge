#!/usr/bin/env python3

import sys

ba = bytearray.fromhex(sys.argv[1])
ba.reverse()
print(ba.hex())

