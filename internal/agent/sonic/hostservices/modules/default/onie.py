"""ONIE boot mode module"""

import logging
from host_modules import host_service
from utils.run_cmd import _run_command

MOD_NAME = 'onie'

ONIE_BOOT_MOUNT = '/mnt/onie-boot'

logger = logging.getLogger(__name__)


class Onie(host_service.HostModule):
    """DBus endpoint that sets ONIE install boot mode"""

    @host_service.method(host_service.bus_name(MOD_NAME), in_signature='', out_signature='is')
    def set_install_mode(self):
        """Mount ONIE-BOOT partition and configure grub to boot into ONIE install mode"""

        rc, _, _ = _run_command('mkdir -p ' + ONIE_BOOT_MOUNT)
        if rc:
            return rc, 'Failed to create mount point ' + ONIE_BOOT_MOUNT

        rc, _, stderr = _run_command('mount LABEL=ONIE-BOOT ' + ONIE_BOOT_MOUNT)
        if rc:
            # Already mounted is acceptable — check /proc/mounts before failing
            already_mounted = False
            try:
                with open('/proc/mounts') as f:
                    already_mounted = any(ONIE_BOOT_MOUNT in line for line in f)
            except OSError:
                pass
            if not already_mounted:
                errmsg = 'Failed to mount ONIE-BOOT: {}'.format(stderr)
                logger.error('%s: %s', MOD_NAME, errmsg)
                return rc, errmsg
            logger.info('%s: ONIE-BOOT already mounted, continuing', MOD_NAME)

        for cmd in [
            'grub-editenv /host/grub/grubenv set next_entry=ONIE',
            'grub-editenv ' + ONIE_BOOT_MOUNT + '/grub/grubenv set onie_mode=install',
        ]:
            rc, _, stderr = _run_command(cmd)
            if rc:
                errmsg = 'Command failed: {!r}, stderr: {}'.format(cmd, stderr)
                logger.error('%s: %s', MOD_NAME, errmsg)
                return rc, errmsg

        logger.info('%s: ONIE install mode set successfully', MOD_NAME)
        return 0, 'ONIE install mode set successfully'


def register():
    """Return the class name"""
    return Onie, MOD_NAME
