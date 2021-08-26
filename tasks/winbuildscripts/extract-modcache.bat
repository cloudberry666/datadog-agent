if exist c:\mnt\modcache.tar (
    Powershell -C "7z e c:\tmp\modcache.tar c:\"
    del /f c:\tmp/modcache.tar
)
