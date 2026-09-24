"""ONIE boot mode module"""

import logging
import subprocess
from host_modules import host_service

MOD_NAME = 'onie'

ONIE_BOOT_MOUNT = '/mnt/onie-boot'

logger = logging.getLogger(__name__)


class Onie(host_service.HostModule):
    """DBus endpoint that sets ONIE install boot mode"""

    @staticmethod
    def _run_command():
        """Mount ONIE-BOOT partition and configure grub to boot into ONIE install mode"""
        cmds = [
            ['mkdir', '-p', ONIE_BOOT_MOUNT],
            ['mount', 'LABEL=ONIE-BOOT', ONIE_BOOT_MOUNT],
            ['grub-editenv', '/host/grub/grubenv', 'set', 'next_entry=ONIE'],
            ['grub-editenv', ONIE_BOOT_MOUNT + '/grub/grubenv', 'set', 'onie_mode=install'],
        ]

        output = ''
        rc = 0
        for cmd in cmds:
            try:
                output = subprocess.check_output(cmd, stderr=subprocess.STDOUT)
            except subprocess.CalledProcessError as err:
                # For the mount command, already mounted is acceptable
                if cmd[0] == 'mount':
                    already_mounted = False
                    try:
                        with open('/proc/mounts') as f:
                            already_mounted = any(ONIE_BOOT_MOUNT in line for line in f)
                    except OSError:
                        pass
                    if already_mounted:
                        logger.info('%s: ONIE-BOOT already mounted, continuing', MOD_NAME)
                        continue
                logger.error('%s: Exception when calling command %s -> %s', MOD_NAME, cmd, err)
                rc = err.returncode
                output = err.output
                return rc, output

        return rc, output

    @host_service.method(host_service.bus_name(MOD_NAME), in_signature='', out_signature='is')
    def set_install_mode(self):
        return Onie._run_command()


def register():
    """Return the class name"""
    return Onie, MOD_NAME
