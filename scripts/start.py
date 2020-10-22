import time
import os
import sys
import urllib
import urllib.request

print('Start')
sys.stdout.flush()
os.chdir('/usr/share/horovod')

def get_init():
    ftp_flag = False
    while not ftp_flag:
        try:
            urllib.request.urlretrieve('http://172.18.29.81/ftp/script/local_init.py', '/usr/share/horovod/local_init.py')
            ftp_flag = True
        except:
            print('FTP error, trying to reconnect...')
            sys.stdout.flush()

def get_params():
    file_path = '/usr/share/horovod/params.tmp'
    s = time.time()
    while True:
        time.sleep(1)
        if os.path.exists(file_path):
            with open(file_path, 'r') as f:
                params = f.readline()
            break
        if time.time() - s > 120:
            raise IOError
    return params

def parse_ip(params):
    ips_end = params.index(' --nodes')
    nodes_end = params.index(' --user_id')
    ips = params[5: ips_end].split(',')[: -1][::-1]
    copy_ssh_id(ips)
    ips = ','.join([ip+':1' for ip in ips])
    print(ips)
    nodes = int(params[ips_end+9: nodes_end])
    return nodes, ips

def copy_ssh_id(ips):
    for ip in ips[1:]:
        os.system('sshpass -p admin123 ssh-copy-id root@%s'%(ip))

def main():
    get_init()

    print('Getting configuration...')
    sys.stdout.flush()
    params = get_params()
    print('Configuraion getted.')
    sys.stdout.flush()
    
    nodes, ips = parse_ip(params)
    os.system('horovodrun --network-interface=net1 --log-level=ERROR -np %d -H %s python -W ignore local_init.py %s'%(nodes, ips, params))

if __name__ == '__main__':
    try:
        main()
        print('Done')
        sys.stdout.flush()
    except:
        print('Err')
        sys.stdout.flush()
