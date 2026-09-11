"""Exercise installers in a temporary filesystem with mocked privileged commands.

No network interfaces, packages, services, firewall or host files are changed.
Windows syntax is additionally checked by the Windows CI job.
"""
import os
from pathlib import Path
import subprocess
import tempfile
import unittest
import zipfile

SOURCE = Path(__file__).resolve().parents[1] / 'server-install.sh'

MOCK = r'''#!/usr/bin/env python3
import os, sys, base64, secrets, pathlib, hashlib
name = pathlib.Path(sys.argv[0]).name
args = sys.argv[1:]
if name == 'id':
    print('0' if args == ['-u'] else 'root')
elif name == 'wg':
    if args[0] in ('genkey', 'genpsk'):
        print(base64.b64encode(secrets.token_bytes(32)).decode())
    elif args[0] == 'pubkey':
        print(base64.b64encode(hashlib.sha256(sys.stdin.buffer.read()).digest()).decode())
elif name == 'wg-quick':
    if args[0] == 'strip': print(pathlib.Path(args[1]).read_text())
elif name == 'systemctl':
    if args[0] == 'is-active': print('active')
elif name == 'curl':
    if os.environ.get('FAIL_HTTPS'): sys.exit(22)
elif name == 'apt-get':
    with open(os.environ['MOCK_LOG'], 'a') as f: f.write('apt-get\n')
'''


class BundleTest(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory()
        self.addCleanup(self.temp.cleanup)
        self.root = Path(self.temp.name)
        self.bin = self.root / 'bin'
        self.bin.mkdir()
        for cmd in ('id', 'wg', 'wg-quick', 'systemctl', 'apt-get', 'curl', 'flock'):
            p = self.bin / cmd
            p.write_text(MOCK)
            p.chmod(0o755)
        self.conf_dir = self.root / 'etc-wireguard'
        source = SOURCE.read_text().replace('/etc/wireguard', str(self.conf_dir))
        source = source.replace('/run/lock', str(self.root))
        self.script = self.root / 'server.sh'
        self.script.write_text(source)
        self.output = self.root / 'bundles'
        self.env = dict(os.environ, PATH=str(self.bin) + os.pathsep + os.environ['PATH'], MOCK_LOG=str(self.root / 'log'))
        self.env.pop('SUDO_USER', None)

    def run_server(self, *args, ok=True):
        result = subprocess.run(['bash', str(self.script), '--endpoint', '203.0.113.10',
                                 '--output-dir', str(self.output), *args], env=self.env,
                                text=True, capture_output=True)
        self.assertEqual(result.returncode == 0, ok, result.stdout + result.stderr)
        return result

    def test_two_devices_preserve_server_and_first_peer(self):
        self.run_server('--client-name', 'win-office')
        config = self.conf_dir / 'wg-newapi.conf'
        first = config.read_text()
        self.run_server('--client-name', 'ubuntu-api', '--add-client')
        second = config.read_text()
        self.assertTrue(second.startswith(first))
        self.assertIn('AllowedIPs = 10.66.0.2/32', second)
        self.assertIn('AllowedIPs = 10.66.0.3/32', second)
        for name in ('win-office', 'ubuntu-api'):
            with zipfile.ZipFile(self.output / (name + '.zip')) as z:
                self.assertEqual(len(z.namelist()), 7)
                client = z.read(name + '.conf').decode()
                self.assertIn('AllowedIPs = 10.66.0.1/32', client)
                self.assertNotIn('0.0.0.0/0', client)
                self.assertIn('Endpoint = 203.0.113.10:51820', client)
                for file in z.namelist():
                    content = z.read(file)
                    self.assertNotIn(b'__TUNNEL_', content)
                    self.assertNotIn(b'__SERVICE_', content)
                    if file.endswith('.ps1'):
                        self.assertTrue(content.startswith(b'\xef\xbb\xbf'))
                    if file.endswith('.sh'):
                        check = subprocess.run(['bash', '-n'], input=content, capture_output=True)
                        self.assertEqual(check.returncode, 0, check.stderr)
            self.assertEqual((self.output / (name + '.zip')).stat().st_mode & 0o777, 0o600)
        self.assertEqual(config.stat().st_mode & 0o777, 0o600)
        self.run_server('--client-name', 'win-office', '--add-client', ok=False)
        self.assertEqual(config.read_text(), second)
        self.run_server('--client-name', 'new-device', ok=False)
        self.assertEqual(config.read_text(), second)

    def test_invalid_addresses_and_names_have_no_side_effects(self):
        for args in (('--client-cidr', '10.66.0.1/32'), ('--client-cidr', '10.67.0.2/32'),
                     ('--server-cidr', '999.1.1.1/24'), ('--client-name', 'abcdefghijklmnop'),
                     ('--endpoint', 'cn.meta-api.vip'), ('--client-name', '../oops'),
                     ('--endpoint',)):
            self.run_server(*args, ok=False)
        self.assertFalse(self.conf_dir.exists())
        self.assertFalse((self.root / 'log').exists())

    def test_failed_https_does_not_create_server(self):
        self.env['FAIL_HTTPS'] = '1'
        self.run_server(ok=False)
        self.assertFalse(self.conf_dir.exists())
        self.assertFalse(self.output.exists())

    def test_duplicate_address_rejected_without_modifying_peers(self):
        self.run_server('--client-name', 'first')
        config = self.conf_dir / 'wg-newapi.conf'
        original = config.read_bytes()
        self.run_server('--add-client', '--client-name', 'second', '--client-cidr', '10.66.0.2/32', ok=False)
        self.assertEqual(config.read_bytes(), original)


if __name__ == '__main__':
    unittest.main()
