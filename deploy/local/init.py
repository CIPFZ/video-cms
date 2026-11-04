#!/usr/bin/env python3
"""Create private, stable configuration for the local Compose deployment."""

import json
from pathlib import Path
import secrets


ROOT = Path(__file__).resolve().parent


def write_private(path, text):
    with path.open('w') as stream:
        path.chmod(0o600)
        stream.write(text)


def main():
    env_file = ROOT / '.env'
    if not env_file.exists():
        values = {
            'CMS_WEB_PORT': '8080',
            'CMS_API_PORT': '18081',
            'CMS_BIND_ADDRESS': '0.0.0.0',
            'MYSQL_PASSWORD': secrets.token_hex(24),
            'MYSQL_ROOT_PASSWORD': secrets.token_hex(24),
            'REDIS_PASSWORD': secrets.token_hex(24),
            'JWT_SIGNING_KEY': secrets.token_hex(32),
            'ADMIN_PASSWORD': 'Cms-' + secrets.token_hex(10) + '!9',
        }
        write_private(env_file, ''.join(f'{k}={v}\n' for k, v in values.items()))
    values = dict(
        line.split('=', 1)
        for line in env_file.read_text().splitlines()
        if line and not line.startswith('#')
    )
    for key in ('MYSQL_PASSWORD', 'MYSQL_ROOT_PASSWORD', 'REDIS_PASSWORD', 'JWT_SIGNING_KEY', 'ADMIN_PASSWORD'):
        if not values.get(key):
            raise SystemExit(f'Missing {key} in {env_file}')

    runtime = ROOT / 'runtime'
    runtime.mkdir(mode=0o700, exist_ok=True)
    config = {
        'system': {'allow_registration': False, 'trusted_proxies': ['172.16.0.0/12', '192.168.0.0/16', '10.0.0.0/8'], 'name': 'VideoCMS', 'environment': 'dev', 'port': 8080,
                   'router_prefix': '/api/v1', 'use_redis': True, 'use_mongo': False},
        'logger': {'level': 'info', 'output': 'stdout', 'format': 'json'},
        'i18n': {'path': 'locales'},
        'jwt': {'signing_key': values['JWT_SIGNING_KEY'], 'expires_time': '7d',
                'buffer_time': '1d', 'issuer': 'nexus-cms'},
        'database': {'driver': 'mysql', 'mysql': {
            'host': 'mysql', 'port': '3306', 'db_name': 'nexus_cms',
            'username': 'nexus_cms', 'password': values['MYSQL_PASSWORD'],
            'max_idle_conns': 5, 'max_open_conns': 30, 'log_mode': 'warn',
        }},
        'redis': {'addr': 'redis:6379', 'password': values['REDIS_PASSWORD'], 'db': 0},
        'file': {'driver': 'local', 'max_mb': 10,
                 'allow_ext': ['.jpg', '.jpeg', '.png', '.gif'],
                 'local': {'path': '/app/uploads/file', 'store_path': '/uploads/file'}},
        'cors': {'mode': 'allow-all'},
        'observable': {'service_name': 'nexus-cms', 'service_version': 'local',
                       'exporter': 'none', 'environment': 'dev'},
        'rate_limit': {'enabled': True, 'qps': 100, 'burst': 200},
    }
    # JSON is valid YAML and avoids requiring a host YAML dependency.
    write_private(runtime / 'backend.yaml', json.dumps(config, indent=2) + '\n')
    write_private(runtime / 'admin.txt',
                  f"URL: http://localhost:{values.get('CMS_WEB_PORT', '8080')}\n"
                  f"Username: admin\nPassword: {values['ADMIN_PASSWORD']}\n")
    print(f'Configuration ready: {env_file}')
    print(f'Admin credentials: {runtime / "admin.txt"}')


if __name__ == '__main__':
    main()
