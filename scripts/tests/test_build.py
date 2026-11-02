"""使用模拟 Docker 验证构建、归档和推送，不构建或推送真实镜像。"""

import json
import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest


ROOT = Path(__file__).resolve().parents[2]
MOCK_DOCKER = r'''#!/usr/bin/env python3
import json, os, sys
from pathlib import Path
args = sys.argv[1:]
path = Path(os.environ['MOCK_STATE'])
state = json.loads(path.read_text())
with open(os.environ['MOCK_LOG'], 'a') as log:
    log.write(json.dumps(args) + '\n')
def done(code=0):
    path.write_text(json.dumps(state))
    sys.exit(code)
def tags(values):
    return [values[i + 1] for i, value in enumerate(values) if value == '--tag']
if args == ['info'] or args == ['buildx', 'version']:
    done()
if args[:2] == ['buildx', 'build']:
    platform = args[args.index('--platform') + 1]
    if os.environ.get('MOCK_FAIL_PLATFORM') == platform:
        done(1)
    state['builds'] += 1
    image = {'id': 'build-' + str(state['builds']), 'platform': platform}
    target = state['remote'] if '--push' in args else state['local']
    for tag in tags(args):
        target[tag] = image
    done()
if args[0] == 'save':
    output = Path(args[args.index('--output') + 1])
    output.write_text(json.dumps({ref: state['local'][ref] for ref in args[3:]}))
    done(1 if os.environ.get('MOCK_SAVE_FAIL') else 0)
if args[0] == 'tag':
    state['local'][args[2]] = state['local'][args[1]]
    done()
if args[0] == 'push':
    if os.environ.get('MOCK_PUSH_FAIL'):
        done(1)
    state['remote'][args[1]] = state['local'][args[1]]
    done()
if args[:3] == ['buildx', 'imagetools', 'create']:
    sources = []
    i = 3
    while i < len(args):
        if args[i] in ('--tag', '--builder'):
            i += 2
        else:
            sources.append(args[i])
            i += 1
    images = {state['remote'][ref]['platform']: state['remote'][ref]['id'] for ref in sources}
    for tag in tags(args):
        state['remote'][tag] = {'images': images, 'sources': sources}
    done()
done(98)
'''


class BuildTests(unittest.TestCase):
    def setUp(self):
        self.temp = tempfile.TemporaryDirectory(prefix='dockerun-build-', dir='/tmp')
        self.addCleanup(self.temp.cleanup)
        self.work = Path(self.temp.name)
        self.root = self.work / 'project files'
        self.root.mkdir()
        self.outside = self.work / 'outside'
        self.outside.mkdir()
        shutil.copy2(ROOT / 'build.sh', self.root / 'build.sh')
        (self.root / 'VERSION').write_text('2.0.0\n')
        (self.root / 'Dockerfile').write_text('FROM scratch\n')
        docker = self.work / 'docker'
        docker.write_text(MOCK_DOCKER)
        docker.chmod(0o755)
        self.log = self.work / 'calls.jsonl'
        self.log.touch()
        self.state_file = self.work / 'state.json'
        self.state_file.write_text(json.dumps({'builds': 0, 'local': {}, 'remote': {}}))
        self.env = {'PATH': str(self.work) + os.pathsep + os.environ['PATH'],
                    'MOCK_STATE': str(self.state_file), 'MOCK_LOG': str(self.log)}

    def run_script(self, *args, success=True, **env):
        result = subprocess.run(['/bin/bash', str(self.root / 'build.sh'), *args], cwd=self.outside,
                                env=self.env | env, capture_output=True, text=True)
        if success:
            self.assertEqual(result.returncode, 0, result.stderr)
            self.assertFalse(result.stderr, result.stderr)
        else:
            self.assertNotEqual(result.returncode, 0, result.stdout)
        return result

    def config(self, **values):
        (self.root / 'build.env').write_text(''.join(f"{key}='{value}'\n" for key, value in values.items()))

    def calls(self):
        return [json.loads(line) for line in self.log.read_text().splitlines()]

    def state(self):
        return json.loads(self.state_file.read_text())

    def archives(self):
        return sorted(self.root.glob('docker-packages/*.tar'))

    def test_default_export_preserves_version_tag_for_each_architecture(self):
        self.run_script()
        self.assertEqual([p.name for p in self.archives()], ['dockerun-2.0.0-amd64.tar', 'dockerun-2.0.0-arm64.tar'])
        for path, platform in zip(self.archives(), ['linux/amd64', 'linux/arm64']):
            self.assertEqual(json.loads(path.read_text())['dockerun:2.0.0']['platform'], platform)
        self.assertEqual(self.state()['remote'], {})
        self.assertEqual(self.state()['builds'], 2)

    def test_configured_single_platform_export_and_push_use_same_built_image(self):
        self.config(REGISTRY='registry.test:5000', IMAGE_NAME='team/app', IMAGE_TAG='3.0.0',
                    PLATFORMS='linux/amd64', EXPORT='1', PUSH='1', EXTRA_TAGS='latest,stable')
        self.run_script()
        archive = json.loads(self.archives()[0].read_text())
        self.assertEqual(set(archive), {'registry.test:5000/team/app:3.0.0', 'registry.test:5000/team/app:latest',
                                        'registry.test:5000/team/app:stable'})
        self.assertEqual(archive, self.state()['remote'])
        self.assertEqual(self.state()['builds'], 1)

    def test_combined_multiplatform_push_assembles_exact_offline_images_without_rebuild(self):
        self.run_script(REGISTRY='registry.test', IMAGE_NAME='team/app', EXPORT='1', PUSH='1',
                        EXTRA_TAGS='latest', NO_CACHE='1', BUILDER='release-builder')
        state = self.state()
        manifest = state['remote']['registry.test/team/app:2.0.0']
        self.assertEqual(state['builds'], 2)
        self.assertEqual(manifest, state['remote']['registry.test/team/app:latest'])
        for path in self.archives():
            image = json.loads(path.read_text())['registry.test/team/app:2.0.0']
            self.assertEqual(manifest['images'][image['platform']], image['id'])
        creates = [a for a in self.calls() if a[:3] == ['buildx', 'imagetools', 'create']]
        self.assertEqual(len(creates), 1)
        self.assertIn('release-builder', creates[0])
        self.assertEqual(creates[0][-2:], ['registry.test/team/app:2.0.0-amd64', 'registry.test/team/app:2.0.0-arm64'])

    def test_legacy_push_without_export_builds_multiplatform_once(self):
        self.run_script(PUSH='1', IMAGE='registry.test:5000/team/app:9.8.7')
        self.assertEqual(self.state()['builds'], 1)
        self.assertEqual(self.archives(), [])
        build = next(a for a in self.calls() if a[:2] == ['buildx', 'build'])
        self.assertIn('--push', build)
        self.assertIn('registry.test:5000/team/app:9.8.7', build)

    def test_full_image_and_environment_override_split_config(self):
        self.config(REGISTRY='unused.test', IMAGE_NAME='unused', IMAGE_TAG='1.0.0', PUSH='1', EXPORT='0')
        self.run_script(IMAGE='registry.test/team/app:4.2.0', PUSH='0', EXPORT='1', PLATFORMS='linux/arm64')
        self.assertEqual(self.archives()[0].name, 'app-4.2.0-arm64.tar')
        self.assertEqual(self.state()['remote'], {})

    def test_custom_builder_context_dockerfile_and_build_flags(self):
        context = self.root / 'custom context'
        context.mkdir()
        (context / 'Dockerfile.prod').write_text('FROM scratch\n')
        self.run_script(PLATFORMS='linux/amd64', BUILD_CONTEXT='custom context', DOCKERFILE='Dockerfile.prod',
                        BUILDER='release-builder', NO_CACHE='1', PULL='1', PROGRESS='plain', OUTPUT_PREFIX='output files/release')
        build = next(a for a in self.calls() if a[:2] == ['buildx', 'build'])
        self.assertEqual(build[-1], str(context))
        for argument in ['release-builder', '--no-cache', '--pull', 'plain', str(context / 'Dockerfile.prod')]:
            self.assertIn(argument, build)
        self.assertTrue((self.root / 'output files/release-amd64.tar').exists())

    def test_preview_does_not_contact_docker_or_create_output(self):
        result = self.run_script('--dry-run', REGISTRY='registry.test', EXPORT='1', PUSH='1')
        self.assertEqual(self.calls(), [])
        self.assertFalse((self.root / 'docker-packages').exists())
        self.assertIn('imagetools create', result.stdout)
        self.assertIn('仅为命令预览', result.stdout)

    def test_help_does_not_load_missing_config(self):
        self.run_script('--help', BUILD_ENV_FILE='/does/not/exist')
        self.assertEqual(self.calls(), [])

    def test_invalid_config_is_rejected_before_docker(self):
        for invalid in [dict(PUSH='true'), dict(PUSH='0', EXPORT='0'), dict(PLATFORMS='linux/amd64,invalid'),
                        dict(PLATFORMS='linux/amd64,linux/amd64'), dict(REGISTRY='https://registry.test'), dict(REGISTRY='registry'),
                        dict(IMAGE_NAME='InvalidName'), dict(IMAGE='app@sha256:1234'), dict(EXTRA_TAGS='latest,'),
                        dict(EXPORT='1', PUSH='1', EXTRA_TAGS='2.0.0-amd64'), dict(DOCKERFILE='missing')]:
            with self.subTest(invalid=invalid):
                self.run_script(success=False, **invalid)
                self.assertEqual(self.calls(), [])

    def test_config_values_are_not_executed(self):
        marker = self.work / 'must-not-exist'
        self.config(PUSH=f'array[$(touch {marker})]')
        self.run_script(success=False)
        self.assertFalse(marker.exists())
        self.assertEqual(self.calls(), [])

    def test_missing_explicit_config_is_not_silently_ignored(self):
        self.run_script(success=False, BUILD_ENV_FILE=str(self.work / 'missing.env'))
        self.assertEqual(self.calls(), [])

    def test_save_failure_keeps_previous_archive_and_cleans_partial(self):
        output = self.root / 'docker-packages'
        output.mkdir()
        archive = output / 'dockerun-2.0.0-amd64.tar'
        archive.write_text('previous archive')
        self.run_script(success=False, MOCK_SAVE_FAIL='1', PLATFORMS='linux/amd64')
        self.assertEqual(archive.read_text(), 'previous archive')
        self.assertEqual(list(output.glob('*.partial.*')), [])

    def test_push_failure_keeps_export_without_claiming_completion(self):
        result = self.run_script(success=False, PUSH='1', EXPORT='1', PLATFORMS='linux/amd64', MOCK_PUSH_FAIL='1')
        self.assertEqual(len(self.archives()), 1)
        self.assertNotIn('构建与交付已完成', result.stdout)

    def test_second_architecture_failure_does_not_publish_partial_index(self):
        self.run_script(success=False, EXPORT='1', PUSH='1', MOCK_FAIL_PLATFORM='linux/arm64')
        self.assertEqual(len(self.archives()), 1)
        self.assertNotIn('dockerun:2.0.0', self.state()['remote'])
        self.assertFalse(any(a[:3] == ['buildx', 'imagetools', 'create'] for a in self.calls()))


if __name__ == '__main__':
    unittest.main()
