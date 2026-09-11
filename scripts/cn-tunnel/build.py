#!/usr/bin/env python3
"""Build public cloud installers from the same templates as offline bundles."""
import argparse
import base64
from pathlib import Path
import re
import shutil

ROOT = Path(__file__).resolve().parent


def build(destination):
    destination = Path(destination)
    destination.mkdir(parents=True, exist_ok=True)
    source = (ROOT / 'server-install.sh').read_text(encoding='utf-8')
    version = (ROOT / 'VERSION').read_text(encoding='utf-8').strip()
    assert f'CN_TUNNEL_VERSION="{version}"' in source
    shutil.copy2(ROOT / 'server-install.sh', destination / 'server-install.sh')
    for suffix, delimiter, encoding in [('sh', 'LINUXCLIENT', 'utf-8'), ('ps1', 'POWERSHELL', 'utf-8-sig')]:
        templates = re.findall(r"<<'" + delimiter + r"'\n(.*?)\n" + delimiter, source, re.S)
        assert len(templates) == 2
        wrapper = (ROOT / f'client-bootstrap.{suffix}.in').read_text(encoding='utf-8')
        for marker, template in zip(('__INSTALL_TEMPLATE_BASE64__', '__UNINSTALL_TEMPLATE_BASE64__'), templates):
            wrapper = wrapper.replace(marker, base64.b64encode((template + '\n').encode(encoding)).decode())
        wrapper = wrapper.replace('__VERSION__', version)
        (destination / f'client-install.{suffix}').write_text(wrapper, encoding=encoding)


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('destination')
    build(parser.parse_args().destination)
