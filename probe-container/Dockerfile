FROM busybox:1.38.0-uclibc

# Enter the namespaces of the VM's PID 1. Requires: --privileged --pid=host
ENTRYPOINT ["nsenter", "-t", "1", "-m", "-u", "-n", "-i", "--"]
CMD ["sh"]
