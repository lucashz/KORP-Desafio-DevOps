# Laboratório Ubuntu no Windows

O WSL 2 executa Linux em uma VM leve gerenciada pelo Windows. Uma distribuição Ubuntu 24.04 dedicada permite instalar Docker Engine e executar o Ansible de verdade, sem depender de servidor externo. O ambiente será identificado como `Korp-Ubuntu-24.04`; ele é separado das outras distribuições e do Docker Desktop.

## Criar o laboratório

Com WSL 2 já instalado, execute no PowerShell:

```powershell
wsl --version
New-Item -ItemType Directory -Force D:\lsk\korp-lab
curl.exe -fL -o D:\lsk\korp-lab\ubuntu-24.04.4-wsl-amd64.wsl https://releases.ubuntu.com/noble/ubuntu-24.04.4-wsl-amd64.wsl
curl.exe -fL -o D:\lsk\korp-lab\SHA256SUMS https://releases.ubuntu.com/noble/SHA256SUMS
Get-FileHash D:\lsk\korp-lab\ubuntu-24.04.4-wsl-amd64.wsl -Algorithm SHA256
Select-String -Path D:\lsk\korp-lab\SHA256SUMS -Pattern wsl-amd64
```

Compare o hash calculado com a entrada do arquivo oficial. Se corresponder, importe:

```powershell
wsl --import Korp-Ubuntu-24.04 D:\lsk\korp-lab\ubuntu D:\lsk\korp-lab\ubuntu-24.04.4-wsl-amd64.wsl --version 2
wsl -d Korp-Ubuntu-24.04 -u root
```

No Ubuntu, confirme `ps -p 1 -o comm=`. Deve mostrar `systemd`. Caso contrário, configure `/etc/wsl.conf` com a seção `[boot]` e `systemd=true`, saia e execute `wsl --terminate Korp-Ubuntu-24.04` no PowerShell antes de entrar novamente.

Não habilite integração do Docker Desktop nessa distribuição: o exercício instala seu próprio Docker Engine. Evite executar as duas cópias da stack simultaneamente, pois ambas usam as portas 80, 3000 e 9090.

## Preparar controlador local

Dentro do Ubuntu:

```bash
apt-get update
apt-get install -y python3-venv ca-certificates
mkdir -p /root/projeto-korp
cp -r /mnt/d/lsk/projeto-korp/. /root/projeto-korp/
cd /root/projeto-korp
python3 -m venv .venv
. .venv/bin/activate
pip install ansible-core==2.19.2
ansible-galaxy collection install -r ansible/requirements.yml
printf '[korp]\nlocalhost ansible_connection=local\n' > ansible/inventory.ini
```

O código fica no filesystem Linux para evitar as permissões e a lentidão de builds diretamente no disco Windows. O uso de root simplifica este laboratório local; no servidor real use usuário SSH comum com sudo.

## Provisionar e demonstrar

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
```

Esse é o único comando de provisionamento depois de preparar o controlador. Ele instala Docker Engine no Ubuntu e sobe os quatro serviços. O destino é `/opt/projeto-korp`; a origem editável no laboratório é `/root/projeto-korp`.

Repita o comando para verificar idempotência. Teste:

```bash
curl http://localhost:80/projeto-korp
cd /opt/projeto-korp
docker compose ps
```

No navegador do Windows, abra `http://localhost:3000`. O WSL normalmente encaminha as portas para localhost. Caso não funcione, confira primeiro `curl http://localhost:3000/api/health` dentro do Ubuntu e possíveis conflitos de portas com o Docker Desktop.

Leia a senha em `/root/projeto-korp/ansible/.secrets/localhost-grafana` apenas quando precisar fazer login; não a exiba durante compartilhamento de tela.

Para abrir o laboratório novamente:

```powershell
wsl -d Korp-Ubuntu-24.04 -u root --cd /root/projeto-korp
```

Ative `.venv/bin/activate` antes de usar Ansible. Para pausar, use `docker compose stop` em `/opt/projeto-korp`; `docker compose start` retoma. `wsl --terminate Korp-Ubuntu-24.04` encerra apenas a distribuição do laboratório. Não use `wsl --unregister` para pausar: esse comando apaga a distribuição.

O WSL demonstra um Docker Engine real em Linux, mas não substitui testes de rede/firewall de uma VM remota. Informe esse contexto ao apresentar a solução.

Referências: [imagem oficial Ubuntu](https://releases.ubuntu.com/noble/), [importação de distribuição no WSL](https://learn.microsoft.com/en-us/windows/wsl/use-custom-distro).
