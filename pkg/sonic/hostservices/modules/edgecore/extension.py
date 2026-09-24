# SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and IronCore contributors
# SPDX-License-Identifier: Apache-2.0

"""Extension module for additional SONiC host operations"""

import os
import re
import subprocess
from datetime import datetime, timezone
from host_modules import host_service

RELOAD_LOCK_FILE = '/etc/sonic/reload.lock'
MOD_NAME = 'extension'
SONIC_ENV_FILE = '/etc/sonic/sonic-environment'


def _parse_reboot_time(raw):
    """Return the reboot time from show reboot-cause output as RFC 3339 (UTC).

    Input example: "User issued 'reboot' command [User: , Time: Tue May  5 09:32:37 AM UTC 2026]"
    Output:        "2026-05-05T09:32:37Z"
    Falls back to the raw time substring if parsing fails.
    """
    m = re.search(r'Time:\s+(.+?)\]', raw)
    if not m:
        return ''
    ts = re.sub(r'\s+', ' ', m.group(1).strip())
    # Drop timezone abbreviation between AM/PM and the year (e.g. "AM UTC 2026" → "AM 2026")
    ts = re.sub(r'(?<=[AP]M)\s+\S+(?=\s+\d{4})', '', ts)
    try:
        dt = datetime.strptime(ts, '%a %b %d %I:%M:%S %p %Y')
        return dt.replace(tzinfo=timezone.utc).strftime('%Y-%m-%dT%H:%M:%SZ')
    except ValueError:
        return m.group(1).strip()


def _read_sonic_env():
    """Parse /etc/sonic/sonic-environment into a dict. Returns (dict, None) or (None, errmsg)."""
    env = {}
    try:
        with open(SONIC_ENV_FILE) as f:
            for line in f:
                line = line.strip()
                if '=' in line and not line.startswith('#'):
                    key, _, val = line.partition('=')
                    env[key.strip()] = val.strip()
    except OSError as e:
        return None, 'failed to read {}: {}'.format(SONIC_ENV_FILE, e)
    return env, None


class Extension(host_service.HostModule):
    """DBus endpoint for additional SONiC host operations"""

    @host_service.method(host_service.bus_name(MOD_NAME), in_signature='', out_signature='is')
    def reboot_cause(self):
        try:
            output = subprocess.check_output(
                ['show', 'reboot-cause'],
                stderr=subprocess.STDOUT,
            )
            return 0, _parse_reboot_time(output.decode('utf-8', errors='replace'))
        except subprocess.CalledProcessError as err:
            return err.returncode, err.output.decode('utf-8', errors='replace').strip()

    @host_service.method(host_service.bus_name(MOD_NAME), in_signature='', out_signature='is')
    def factory_reset(self):
        subprocess.Popen(
            ['sudo', 'reset-factory'],
            shell=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
            start_new_session=True,
        )
        return 0, ""

    @host_service.method(host_service.bus_name(MOD_NAME), in_signature='', out_signature='is')
    def reprovision(self):
        if os.path.exists(RELOAD_LOCK_FILE):
            return 2, 'reload already in progress: {} exists'.format(RELOAD_LOCK_FILE)

        env, err = _read_sonic_env()
        if err:
            return 1, err

        platform = env.get('PLATFORM', '')
        hwsku = env.get('HWSKU', '')
        if not platform or not hwsku:
            return 1, 'PLATFORM or HWSKU not found in {}'.format(SONIC_ENV_FILE)

        commands = [
            ['bash', '-c',
             'set -o pipefail; sudo sonic-cfggen --preset l1 -k {} -p /usr/share/sonic/device/{}/{}/port_config.ini'
             ' | sudo tee /etc/sonic/config_db.json > /dev/null'.format(hwsku, platform, hwsku)],
            ['bash', '-c',
             "jq '.DEVICE_METADATA.localhost += {\"docker_routing_config_mode\": \"split-unified\"}"
             " | .DEVICE_METADATA.localhost.type = \"ToRRouter\""
             " | del(.MGMT_VRF_CONFIG)'"
             " /etc/sonic/config_db.json > /tmp/config_db.json && sudo mv /tmp/config_db.json /etc/sonic/config_db.json"],
            ['sudo', 'config', 'reload', '/etc/sonic/config_db.json', '-y','-f'],
        ]

        for cmd in commands:
            try:
                subprocess.check_output(cmd, stderr=subprocess.STDOUT)
            except subprocess.CalledProcessError as e:
                errmsg = 'command {!r} failed: {}'.format(
                    cmd[1], e.output.decode('utf-8', errors='replace').strip())
                return e.returncode, errmsg

        return 0, ''


def register():
    """Return class and module name"""
    return Extension, MOD_NAME
