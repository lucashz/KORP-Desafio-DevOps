# Laboratório Rocky Linux no VirtualBox

O playbook possui instalação específica para Rocky Linux 9/10. A versão exata deve ser confirmada com `cat /etc/rocky-release` antes do provisionamento. A instalação usa DNF, chave de assinatura verificada e repositório Docker CE indicado pela documentação do Rocky.

## Acesso inicial

No console da VM, como root:

```bash
cat /etc/rocky-release
ip -br address
dnf install -y openssh-server
systemctl enable --now sshd
systemctl status sshd --no-pager
```

Se o firewalld estiver ativo e SSH ainda não estiver permitido na zona da interface, habilite o serviço SSH nessa zona. Confira com `firewall-cmd --get-active-zones` e `firewall-cmd --list-services`; não desative o firewall para resolver o acesso.

No VirtualBox, o adaptador em modo bridge permite que a VM tenha um IP na mesma rede do Windows. Confirme o endereço com `ip -br address`; ele pode mudar se for obtido por DHCP. No PowerShell, teste `ssh root@IP_DA_VM` e informe a senha quando solicitado. A senha não deve ser incluída no inventário.

## Controlador na própria VM

Copie o projeto para `/root/projeto-korp`, sem `.env`, `.secrets`, `.venv` e inventários reais. No Rocky 9, instale Python 3.12 para executar Ansible 2.19; o Python padrão do sistema pode ser mais antigo:

```bash
dnf install -y python3.12 python3.12-pip
cd /root/projeto-korp
python3.12 -m venv .venv
. .venv/bin/activate
pip install ansible-core==2.19.2
ansible-galaxy collection install -r ansible/requirements.yml
printf '[korp]\nlocalhost ansible_connection=local ansible_python_interpreter=/usr/bin/python3.12\n' > ansible/inventory.ini
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml
```

No Rocky 10, use o Python 3.12 disponibilizado pela distribuição. Os módulos DNF precisam também das bibliotecas Python do gerenciador de pacotes do sistema; o Ansible pode executar essa parte com o interpretador do sistema. A execução real do playbook é necessária para confirmar a compatibilidade na versão instalada.

## SELinux e rede

Os bind mounts de configuração usam `ro,Z`: leitura apenas e rótulo SELinux privado para o container correspondente. Isso permite manter SELinux enforcing. Não é necessário usar `setenforce 0`. Os diretórios montados pertencem ao projeto; não aplique relabel em diretórios de sistema inteiros.

Docker publica NGINX na porta 80. Grafana e Prometheus ficam em loopback. Do Windows, abra um túnel:

```powershell
ssh -N -L 3000:127.0.0.1:3000 -L 9090:127.0.0.1:9090 root@IP_DA_VM
```

Abra `http://localhost:3000` no navegador. A senha inicial fica em `/root/projeto-korp/ansible/.secrets/localhost-grafana` se o controlador for a própria VM. Para a API, use `http://IP_DA_VM/projeto-korp`.

Docker administra regras próprias de encaminhamento. Verifique o acesso pela rede real após subir a stack; uma chamada feita dentro da VM não comprova acesso pelo Windows. O provisionamento não altera políticas gerais do firewall nem desativa SELinux.

Referência: [Docker Engine no Rocky Linux](https://docs.rockylinux.org/gemstones/containers/docker/).
