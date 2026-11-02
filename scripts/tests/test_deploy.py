"""使用模拟 Docker 验证部署顺序及失败保护，不操作真实容器或数据库。"""

import json
import os
from pathlib import Path
import socket
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
MOCK_DOCKER = r'''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
state_path = Path(os.environ['MOCK_STATE'])
state = json.loads(state_path.read_text())
with open(os.environ['MOCK_LOG'], 'a') as log:
    log.write(json.dumps({'args': args, 'dsn': os.environ.get('MYSQL_DSN'), 'url': os.environ.get('PUBLIC_URL')}) + '\n')
containers = state['containers']
def done(code=0, output=''):
    state_path.write_text(json.dumps(state))
    if output:
        print(output)
    sys.exit(code)
if args == ['info']:
    done()
if args[:2] == ['image', 'inspect']:
    if '--format' in args and args[-1] in state['images']:
        done(output=os.environ.get('MOCK_IMAGE_PLATFORM', 'linux/amd64'))
    done(0 if args[-1] in state['images'] else 1)
if args[0] == 'version':
    done(output='linux/amd64')
if args[:2] == ['network', 'inspect']:
    done(1 if args[-1] == os.environ.get('MOCK_MISSING_NETWORK') else 0)
if args[:2] == ['network', 'connect']:
    done(1 if os.environ.get('MOCK_CONNECT_FAIL') else 0)
if args[:2] == ['container', 'inspect']:
    container = containers.get(args[-1])
    if container is None:
        done(1)
    if '--format' not in args:
        done()
    template = args[args.index('--format') + 1]
    if template == '{{.State.Running}}':
        done(output='true' if container['running'] else 'false')
    status = 'running' if container['running'] else 'exited'
    health = os.environ.get('MOCK_HEALTH', 'healthy')
    done(output=status + ' ' + health)
if args[0] == 'create':
    if os.environ.get('MOCK_CREATE_FAIL'):
        done(1)
    name = args[args.index('--name') + 1]
    containers[name] = {'running': False, 'image': args[-1]}
    done(output='candidate-id')
if args[0] == 'rename':
    if args[-1] in containers:
        done(1)
    containers[args[-1]] = containers.pop(args[-2])
    done()
if args[0] in ('stop', 'start', 'restart'):
    if args[0] == 'start' and os.environ.get('MOCK_START_FAIL'):
        done(1)
    containers[args[-1]]['running'] = args[0] != 'stop'
    done()
if args[0] == 'rm':
    containers.pop(args[-1], None)
    done()
if args[0] == 'pull':
    state['images'].append(args[-1])
    done()
if args[:2] == ['image', 'rm']:
    done()
if args[0] in ('port', 'logs', 'exec'):
    done()
done(98, '未实现的模拟命令：' + repr(args))
'''


class DeployTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='dockerun-deploy-', dir='/tmp')
        self.addCleanup(self.temp.cleanup)
        self.work = Path(self.temp.name)
        self.socket = socket.socket(socket.AF_UNIX)
        self.socket.bind(str(self.work / 'docker.sock'))
        self.addCleanup(self.socket.close)
        self.config = self.work / 'deploy.env'
        self.state_file = self.work / 'state.json'
        self.log_file = self.work / 'calls.jsonl'
        self.log_file.touch()
        docker = self.work / 'docker'
        docker.write_text(MOCK_DOCKER)
        docker.chmod(0o755)
        self.env = {
            'PATH': str(self.work) + os.pathsep + os.environ['PATH'],
            'CONFIG_FILE': str(self.config),
            'MOCK_STATE': str(self.state_file),
            'MOCK_LOG': str(self.log_file),
        }
        self.values = {
            'CONTAINER_NAME': 'dockerun',
            'IMAGE_NAME': 'dockerun:2.0.0',
            'DOCKER_NETWORK': 'ingot-net',
            'DATA_DIR': str(self.work / 'data with spaces'),
            'DOCKER_SOCKET': str(self.work / 'docker.sock'),
            'DATABASE_TYPE': 'mysql',
            'MYSQL_DSN': 'fixture:pa$word@tcp(mysql.test:3306)/fixture',
            'PUBLIC_URL': 'https://dockerun.example.com',
            'VIRTUAL_HOST': 'dockerun.example.com',
            'LETSENCRYPT_HOST': 'dockerun.example.com',
            'HOST_PORT': '19000',
            'PULL_POLICY': 'never',
            'STARTUP_TIMEOUT': '1',
        }
        self.write_config()
        self.set_state()

    def write_config(self):
        self.config.write_text(''.join(f"{key}='{value}'\n" for key, value in self.values.items()))

    def set_state(self, existing=False, image=True):
        self.state_file.write_text(json.dumps({
            'images': ['dockerun:2.0.0'] if image else [],
            'containers': {'dockerun': {'running': True, 'image': 'dockerun:1.0.0'}} if existing else {},
        }))

    def run_script(self, action, success=True, **env):
        result = subprocess.run(['/bin/bash', str(ROOT / 'deploy.sh'), action], cwd=self.work,
                                env=self.env | env, capture_output=True, text=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0)
        return result

    def calls(self):
        return [json.loads(line) for line in self.log_file.read_text().splitlines()]

    def state(self):
        return json.loads(self.state_file.read_text())['containers']

    def assert_old_untouched(self):
        self.assertEqual(self.state()['dockerun'], {'running': True, 'image': 'dockerun:1.0.0'})
        self.assertFalse(any(call['args'][0] in ('stop', 'rename', 'start') for call in self.calls()))

    def test_help_needs_neither_config_nor_docker(self):
        self.config.unlink()
        self.run_script('help')
        self.assertEqual(self.calls(), [])

    def test_check_is_read_only_even_with_always_pull(self):
        self.values['PULL_POLICY'] = 'always'
        self.write_config()
        self.set_state(existing=True)
        self.run_script('check')
        self.assert_old_untouched()
        self.assertFalse(Path(self.values['DATA_DIR']).exists())
        self.assertFalse(any(c['args'][0] in ('pull', 'create', 'rm') for c in self.calls()))

    def test_first_start_passes_literal_secrets_without_command_arguments(self):
        marker = self.work / 'must-not-exist'
        self.values['MYSQL_DSN'] += f'$(touch {marker})'
        self.write_config()
        result = self.run_script('start')
        create = next(c for c in self.calls() if c['args'][0] == 'create')
        self.assertEqual(create['dsn'], self.values['MYSQL_DSN'])
        self.assertNotIn(self.values['MYSQL_DSN'], ' '.join(create['args']) + result.stdout + result.stderr)
        self.assertFalse(marker.exists())
        self.assertIn(f"type=bind,src={self.values['DATA_DIR']},dst={self.values['DATA_DIR']}", create['args'])
        self.assertIn('127.0.0.1:19000:8080', create['args'])
        self.assertIn('VIRTUAL_PORT=8080', create['args'])
        self.assertEqual(Path(self.values['DATA_DIR']).stat().st_mode & 0o777, 0o700)
        self.assertTrue(self.state()['dockerun']['running'])

    def test_start_of_existing_container_does_not_silently_replace_it(self):
        self.set_state(existing=True)
        result = self.run_script('start')
        self.assert_old_untouched()
        self.assertIn('update', result.stdout)

    def test_update_prepares_candidate_and_networks_before_stopping_old(self):
        self.set_state(existing=True)
        self.values['ADDITIONAL_NETWORKS'] = 'mysql-net'
        self.write_config()
        self.run_script('update')
        calls = [call['args'] for call in self.calls()]
        self.assertLess(next(i for i, a in enumerate(calls) if a[:2] == ['network', 'connect']),
                        next(i for i, a in enumerate(calls) if a[0] == 'stop'))
        self.assertEqual(self.state()['dockerun']['image'], 'dockerun:2.0.0')
        previous = [v for k, v in self.state().items() if k.startswith('dockerun-previous-')]
        self.assertEqual(previous, [{'running': False, 'image': 'dockerun:1.0.0'}])
        self.assertFalse(any(a[0] == 'rm' for a in calls))

    def test_missing_local_image_keeps_old_container_running(self):
        self.set_state(existing=True, image=False)
        self.run_script('update', success=False)
        self.assert_old_untouched()
        self.assertFalse(any(c['args'][0] == 'pull' for c in self.calls()))

    def test_missing_network_keeps_old_container_running(self):
        self.set_state(existing=True)
        self.run_script('update', success=False, MOCK_MISSING_NETWORK='ingot-net')
        self.assert_old_untouched()

    def test_wrong_architecture_keeps_old_container_running(self):
        self.set_state(existing=True)
        result = self.run_script('update', success=False, MOCK_IMAGE_PLATFORM='linux/arm64')
        self.assertIn('平台', result.stderr)
        self.assert_old_untouched()

    def test_missing_policy_can_pull_before_replacing_old(self):
        self.set_state(existing=True, image=False)
        self.run_script('update', PULL_POLICY='missing')
        calls = [c['args'] for c in self.calls()]
        self.assertLess(next(i for i, a in enumerate(calls) if a[0] == 'pull'),
                        next(i for i, a in enumerate(calls) if a[0] == 'stop'))

    def test_invalid_proxy_domain_is_rejected_before_changes(self):
        self.set_state(existing=True)
        self.run_script('update', success=False, PUBLIC_URL='https://other.example.com')
        self.assert_old_untouched()

    def test_invalid_mount_and_port_are_rejected_before_changes(self):
        self.set_state(existing=True)
        for invalid in ({'DATA_DIR': '/tmp/..'}, {'HOST_PORT': '65536'},
                        {'HOST_BIND_IP': '999.0.0.1'}, {'RESTART_POLICY': 'on-failure:1bad'}):
            with self.subTest(invalid=invalid):
                self.run_script('update', success=False, **invalid)
                self.assert_old_untouched()

    def test_failed_create_keeps_old_container_running(self):
        self.set_state(existing=True)
        self.run_script('update', success=False, MOCK_CREATE_FAIL='1')
        self.assert_old_untouched()

    def test_failed_extra_network_cleans_only_candidate(self):
        self.set_state(existing=True)
        self.values['ADDITIONAL_NETWORKS'] = 'mysql-net'
        self.write_config()
        self.run_script('update', success=False, MOCK_CONNECT_FAIL='1')
        self.assert_old_untouched()
        self.assertEqual(list(self.state()), ['dockerun'])

    def test_failed_health_does_not_restart_old_after_possible_migration(self):
        self.set_state(existing=True)
        result = self.run_script('update', success=False, MOCK_HEALTH='unhealthy')
        self.assertIn('未自动启动旧版本', result.stderr)
        self.assertTrue(self.state()['dockerun']['running'])
        previous = [v for k, v in self.state().items() if k.startswith('dockerun-previous-')]
        self.assertEqual(previous, [{'running': False, 'image': 'dockerun:1.0.0'}])
        self.assertFalse(any(c['args'][0] == 'start' and '-previous-' in c['args'][-1] for c in self.calls()))

    def test_exported_variables_override_file_including_empty(self):
        self.run_script('start', HOST_PORT='', MYSQL_DSN='fixture:override@tcp(mysql.test:3306)/fixture')
        create = next(c for c in self.calls() if c['args'][0] == 'create')
        self.assertNotIn('--publish', create['args'])
        self.assertEqual(create['dsn'], 'fixture:override@tcp(mysql.test:3306)/fixture')

    def test_empty_mysql_dsn_is_rejected_before_changes(self):
        self.set_state(existing=True)
        result = self.run_script('update', success=False, MYSQL_DSN='')
        self.assertIn('MYSQL_DSN', result.stderr)
        self.assert_old_untouched()

    def test_unknown_config_does_not_override_system_environment(self):
        with self.config.open('a') as file:
            file.write('PATH=/must-not-be-used\n')
        self.run_script('check', success=False)
        self.assertEqual(self.calls(), [])

    def test_restart_preserves_existing_configuration_without_pulling(self):
        self.set_state(existing=True)
        self.run_script('restart')
        self.assertFalse(any(c['args'][0] in ('create', 'pull', 'rename') for c in self.calls()))
        self.assertEqual(self.state()['dockerun']['image'], 'dockerun:1.0.0')

    def test_remove_image_does_not_stop_or_remove_container(self):
        self.set_state(existing=True)
        self.run_script('remove-image')
        self.assert_old_untouched()
        self.assertFalse(any(c['args'][0] == 'rm' for c in self.calls()))


if __name__ == '__main__':
    unittest.main()
