# Provisionamento

O controlador precisa de Python 3.11 ou superior e acesso SSH ao destino. O destino precisa de Python compatível com Ansible 2.19, acesso à internet e usuário com sudo ou root. As tarefas de instalação ficam em `tasks/docker-rocky.yml` e `tasks/docker-ubuntu.yml`.

## Preparar o controlador

Na raiz do repositório:

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install ansible-core==2.19.2
ansible-galaxy collection install -r ansible/requirements.yml
cp ansible/inventory.example.ini ansible/inventory.ini
```

No Ubuntu, pode ser necessário instalar `python3-venv`. No Rocky 9, use Python 3.12 para criar o ambiente virtual; o Python padrão pode ser mais antigo. Confirme a versão com `python3 --version`. As bibliotecas DNF do sistema também precisam estar disponíveis para as tarefas de instalação.

Edite o inventário com o endereço e usuário do destino. Não inclua senhas. Para autenticação SSH por senha, use `--ask-pass`; para sudo por senha, use `--ask-become-pass`.

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --ask-become-pass
```

Para executar na própria máquina Linux, use este inventário:

```ini
[korp]
localhost ansible_connection=local
```

Quando necessário, defina `ansible_python_interpreter` com o caminho do Python do destino. Ao executar como root, omita `--ask-become-pass`.

A porta do Grafana é 3000 por padrão. Se já estiver ocupada, acrescente `-e grafana_port=3001` ao comando do playbook. Para testar essa porta com `scripts/smoke.py`, exporte `GRAFANA_URL=http://localhost:3001`.

## Arquivos e acesso

O deploy fica em `/opt/projeto-korp`. A senha do Grafana é gerada e preservada em `ansible/.secrets/<host>-grafana` no controlador. O `.env` do destino recebe permissão `0600`. Inventário real, `.env` e `.secrets` são ignorados pelo Git.

A senha inicial do Grafana é usada na criação do banco. Alterar apenas o `.env` não troca a senha de um usuário já existente; preserve o arquivo de senha entre execuções.

Grafana publica a porta 3000 no destino. Acesse `http://IP_DO_SERVIDOR:3000` e entre como `admin`. A API fica em `http://IP_DO_SERVIDOR/projeto-korp`. O firewall da rede deve permitir o acesso à porta 3000.

Prometheus escuta somente no loopback. Para acessar sua interface remotamente:

```bash
ssh -N -L 9090:127.0.0.1:9090 usuario@IP_DO_SERVIDOR
```

Abra `http://localhost:9090` no navegador.

## Reexecução

O playbook mantém pacotes e serviços existentes. Quando algum arquivo de deploy muda, reconstrói a imagem e recria os serviços para aplicar as configurações; isso pode causar uma breve interrupção.

A validação termina com uma chamada HTTP pelo NGINX, conferência dos campos JSON, consulta ao Prometheus e consulta autenticada ao dashboard. Repita o playbook no destino para verificar idempotência. O provisionamento completo no Rocky Linux ainda não foi validado.

Referências: [Docker no Rocky Linux](https://docs.rockylinux.org/gemstones/containers/docker/), [Docker no Ubuntu](https://docs.docker.com/engine/install/ubuntu/) e [módulo Compose V2](https://docs.ansible.com/projects/ansible/latest/collections/community/docker/docker_compose_v2_module.html).
