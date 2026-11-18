"""Check the editor's interface text: no Chinese left hard-coded in app.js,
and every key the page uses exists in both languages."""
import re, sys, io

root = sys.argv[1] if len(sys.argv) > 1 else '.'
app = io.open(root + '/pkg/editor/web/app.js', encoding='utf-8').read()
i18n = io.open(root + '/pkg/editor/web/i18n.js', encoding='utf-8').read()
html = io.open(root + '/pkg/editor/web/index.html', encoding='utf-8').read()

bad = 0
in_comment = False
for n, line in enumerate(app.splitlines(), 1):
    s = line.strip()
    if in_comment:
        if '*/' in s:
            in_comment = False
        continue
    if s.startswith('/*'):
        in_comment = '*/' not in s
        continue
    if s.startswith('*'):
        continue
    code = line.split('//')[0]
    if re.search(r'[一-鿿]', code):
        print('hard-coded Chinese, app.js line', n, ':', line.strip()[:100])
        bad += 1

# Whole keys only: t('el.' + id) builds its key at run time; those are listed
# below by prefix.
keys = {k for k in re.findall(r"\bt\('([a-zA-Z0-9.\-]+)'", app) if not k.endswith('.')}
keys |= set(re.findall(r'data-i18n(?:-title|-placeholder)?="([^"]+)"', html))
# keys built at run time: 'el.' + id, 'tool.' + id, 'stamp.' + id, 'unit.' + id, 'kind.' + k, 'look.' + field, 'err.' + code
for prefix, ids in [
    ('el.', 'ground brick question stair coin goomba koopa hidden spawn flag'),
    ('tool.', 'pen rect erase pick select stamp'),
    ('stamp.', 'pipe stairs-up stairs-down gap brick-row coin-arc'),
    ('unit.', 'pipe stairs-up stairs-down gap brick-row coin-arc'),
    ('kind.', 'coin mushroom star oneup poison gold random'),
    ('look.', 'turf ground brick stair pipe'),
    ('err.', 'bad-id-chars bad-id-form read-only invalid-level write-failed bad-request bad-source unknown'),
]:
    keys |= {prefix + i for i in ids.split()}

zh = i18n[i18n.index('  zh: {'):i18n.index('  en: {')]
en = i18n[i18n.index('  en: {'):]
for k in sorted(keys):
    for name, block in (('zh', zh), ('en', en)):
        if "'" + k + "'" not in block:
            print('missing', name, k)
            bad += 1
zh_keys = set(re.findall(r"'([a-zA-Z0-9.\-]+)':", zh))
en_keys = set(re.findall(r"'([a-zA-Z0-9.\-]+)':", en))
for k in sorted(zh_keys ^ en_keys):
    print('only in one language:', k)
    bad += 1
print('keys used:', len(keys), '| problems:', bad)
sys.exit(1 if bad else 0)
