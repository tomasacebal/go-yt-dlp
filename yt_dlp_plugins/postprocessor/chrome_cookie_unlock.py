"""
Plugin local para yt-dlp.

Objetivo:
- Evitar fallos de "Permission denied" al copiar la DB de cookies de Chromium
  cuando el navegador mantiene lock en Windows.

Fuente:
- Basado en seproDev/yt-dlp-ChromeCookieUnlock (MIT)
- Basado en gist de Charles Machalow
"""

import sys
import importlib
from ctypes import (
    WINFUNCTYPE,
    byref,
    create_unicode_buffer,
    pointer,
    windll,
)
from ctypes.wintypes import DWORD, UINT, WCHAR

cookies = importlib.import_module("yt_dlp.cookies")


ERROR_SUCCESS = 0
ERROR_MORE_DATA = 234
RM_FORCE_SHUTDOWN = 1


@WINFUNCTYPE(None, UINT)
def callback(_percent_complete: UINT) -> None:
    return None


rstrtmgr = windll.LoadLibrary("Rstrtmgr")
original_open_database_copy = yt_dlp.cookies._open_database_copy


def unlock_cookies(cookies_path: str) -> None:
    session_handle = DWORD(0)
    session_flags = DWORD(0)
    session_key = (WCHAR * 256)()

    result = DWORD(
        rstrtmgr.RmStartSession(byref(session_handle), session_flags, session_key)
    ).value
    if result != ERROR_SUCCESS:
        raise RuntimeError(f"RmStartSession returned non-zero result: {result}")

    try:
        result = DWORD(
            rstrtmgr.RmRegisterResources(
                session_handle,
                1,
                byref(pointer(create_unicode_buffer(cookies_path))),
                0,
                None,
                0,
                None,
            )
        ).value
        if result != ERROR_SUCCESS:
            raise RuntimeError(f"RmRegisterResources returned non-zero result: {result}")

        proc_info_needed = DWORD(0)
        proc_info = DWORD(0)
        reboot_reasons = DWORD(0)
        result = DWORD(
            rstrtmgr.RmGetList(
                session_handle,
                byref(proc_info_needed),
                byref(proc_info),
                None,
                byref(reboot_reasons),
            )
        ).value
        if result not in (ERROR_SUCCESS, ERROR_MORE_DATA):
            raise RuntimeError(f"RmGetList returned non-successful result: {result}")

        if proc_info_needed.value:
            result = DWORD(
                rstrtmgr.RmShutdown(session_handle, RM_FORCE_SHUTDOWN, callback)
            ).value
            if result != ERROR_SUCCESS:
                raise RuntimeError(f"RmShutdown returned non-successful result: {result}")
    finally:
        result = DWORD(rstrtmgr.RmEndSession(session_handle)).value
        if result != ERROR_SUCCESS:
            raise RuntimeError(f"RmEndSession returned non-successful result: {result}")


def unlock_chrome(database_path, tmpdir):
    try:
        return original_open_database_copy(database_path, tmpdir)
    except PermissionError:
        print("Attempting to unlock cookies", file=sys.stderr)
        unlock_cookies(database_path)
        return original_open_database_copy(database_path, tmpdir)


yt_dlp.cookies._open_database_copy = unlock_chrome

