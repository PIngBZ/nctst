#coding=utf-8

import sys
import os
import re
import json

SelectPerGroup = 2
ConnNumPerServer= 3

# generator.py output.json in1.yaml in2.yaml in....yaml
if len(sys.argv) < 3:
    print("params error")
    sys.exit(-1)

if os.path.exists(sys.argv[1]):
    print("output file exist")
    sys.exit(-2)

def readFile(file):
    with open(file, 'r') as f:
        return f.read()

def src2group(file):
    content = readFile(file)
    group = {'name': file, 'list': []}
    p = re.compile(r'\{\s*?(name.*?)\}', re.DOTALL)
    r = p.findall(content)

    for m in r:
        proxy = {'connnum': ConnNumPerServer}
        params = {}

        items = m.split(',')
        for item in items:

            kv = item.split(':', 1)
            k, v = kv[0].strip(), kv[1].strip()

            if k == 'type':
                proxy['type'] = v
            elif k == 'server':
                proxy['host'] = v
            elif k == 'port':
                proxy['port'] = int(v)
            elif k == 'password':
                proxy['password'] = v
            else:
                params[k] = v

        proxy['params'] = params
        group['list'].append(proxy)

    if len(group['list']) == 0:
        return None
    return group

result = { 'selectpergroup': SelectPerGroup, 'groups': []}

for f in sys.argv[2:]:
    group = src2group(f)
    if not group:
        sys.exit(-2)
    result['groups'].append(group)


with open(sys.argv[1], 'w') as out:
    out.write(json.dumps(result, indent = 4))



