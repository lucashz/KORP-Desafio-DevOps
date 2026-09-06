# Projeto Korp

Serviço HTTP em Go com proxy reverso NGINX, monitoramento Prometheus/Grafana e provisionamento Ansible para Rocky Linux 9/10 ou Ubuntu 24.04.

`GET /projeto-korp` retorna dois campos, com horário UTC calculado a cada requisição:

```json
{"nome":"Projeto Korp","horario":"2026-09-05T20:30:00.123456789Z"}
```

## Arquitetura

```text
Cliente → host:80 → NGINX → http-server-projeto-korp:8080
                                    ↑ /metrics
                                Prometheus ← Grafana
```

Os quatro containers usam a rede bridge `projeto-korp-network`. A aplicação não publica portas no host. NGINX publica a porta 80; Grafana e Prometheus publicam 3000 e 9090 somente em loopback. O NGINX bloqueia `/metrics`; o Prometheus coleta diretamente pelo DNS interno do Docker.

## Provisionamento completo com Ansible

Destino: Rocky Linux 9/10 ou Ubuntu 24.04, amd64 ou arm64, com Python compatível, acesso à internet e usuário com sudo ou root. Controlador: Linux com Python 3.11 ou superior, SSH e acesso ao destino. Recomenda-se uma VM dedicada com 2 vCPU, 4 GB de RAM e 15 GB livres. O controlador pode ser a própria VM. Consulte o [laboratório Rocky Linux no VirtualBox](docs/AMBIENTE-ROCKY.md) para preparar o Python e o acesso SSH.

Prepare as dependências do controlador uma vez, na raiz do projeto:

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install ansible-core==2.19.2
ansible-galaxy collection install -r ansible/requirements.yml
cp ansible/inventory.example.ini ansible/inventory.ini
```

Se necessário, instale `python3-venv` no controlador. Edite o inventário com IP e usuário SSH reais. Para execução na própria VM, use `localhost ansible_connection=local` dentro do grupo `[korp]`. Confira previamente a chave do servidor com uma conexão SSH normal.

Provisione tudo com um comando:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --ask-become-pass
```

O playbook instala Docker pelo repositório oficial, habilita o serviço, copia apenas os arquivos de deploy para `/opt/projeto-korp`, gera uma senha persistente, cria a rede, constrói a imagem, inicia os quatro containers e valida API, coleta e dashboard. A resposta JSON aparece no console. Uma segunda execução sem alterações deve manter os recursos existentes.

O inventário real e `ansible/.secrets/` não são versionados. A senha do Grafana fica em `ansible/.secrets/<nome-do-host>-grafana` no controlador e no `.env` remoto com permissão `0600`. Preserve esse arquivo: a senha inicial não é redefinida automaticamente quando o Grafana já possui um volume de dados.

Para acessar um servidor remoto pelo navegador:

```bash
ssh -L 3000:127.0.0.1:3000 -L 9090:127.0.0.1:9090 ubuntu@IP_DO_SERVIDOR
```

Abra [Grafana](http://localhost:3000), entre como `admin` com a senha gerada e acesse **Dashboards → Projeto Korp → Projeto Korp — Serviço HTTP**. O dashboard é provisionado por arquivo, sem importação manual.

## Execução local com Docker já instalado

Na raiz do projeto, crie `.env` a partir de `.env.example` e substitua o valor por uma senha própria. Depois:

```bash
docker network create --driver bridge projeto-korp-network
docker compose up --build -d --wait
curl http://localhost:80/projeto-korp
```

Se a rede já existir, reutilize-a. A rede é externa ao Compose porque a criação explícita faz parte do provisionamento Ansible. No PowerShell, use `curl.exe` para evitar o alias de versões antigas.

| Endereço | Finalidade |
| --- | --- |
| `http://localhost:80/projeto-korp` | API pelo proxy |
| `http://localhost:80/healthz` | Saúde da aplicação pelo proxy |
| `http://localhost:3000` | Grafana com autenticação |
| `http://localhost:9090` | Interface local do Prometheus |

## Monitoramento

- `up{job="http-server-projeto-korp"}`: 1 quando a coleta funciona, 0 quando falha. É uma medida de disponibilidade do alvo de coleta, não de toda a experiência do cliente.
- `http_requests_total{method,route,status}`: contador de requisições da aplicação, incluindo 404 e 405; exclui `/healthz` e `/metrics`.
- `http_request_duration_seconds`: histograma de duração do handler; não inclui rede ou proxy.
- Métricas padrão do runtime Go e do processo também estão disponíveis.

O dashboard inclui disponibilidade atual/histórica, percentual de sucesso das coletas em 1h, volume estimado no período, requisições por segundo, volume por status, p95 e latência média. Use `sh scripts/load.sh` para gerar 300 requisições ao longo de aproximadamente um minuto. Antes de existir tráfego suficiente, gráficos de latência podem mostrar ausência de dados.

A regra `KorpServiceDown` dispara no Prometheus após 30 segundos com `up == 0`. Não há Alertmanager nem envio de notificações configurado.

## Verificação

Com Go 1.26 instalado:

```bash
go test -race -cover ./...
go vet ./...
```

Com a stack ativa:

```bash
docker compose exec -T nginx nginx -t
docker compose exec -T prometheus promtool check config /etc/prometheus/prometheus.yml
export GRAFANA_ADMIN_PASSWORD='sua-senha'
python3 scripts/smoke.py
```

O teste integrado verifica o contrato JSON, atualização do horário UTC, bloqueio externo de métricas, scrape e dashboard. O CI executa testes Go, sintaxe Ansible e integração com os quatro containers. A instalação completa em Linux deve ser demonstrada conforme [o roteiro](docs/DEMONSTRACAO.md); validação de sintaxe não comprova provisionamento.

## Operação

```bash
docker compose ps
docker compose logs --tail=100 http-server-projeto-korp nginx
docker compose stop http-server-projeto-korp
docker compose start http-server-projeto-korp
docker compose down
```

`down` preserva os volumes nomeados e a rede externa. `down -v` apaga os dados do Grafana e Prometheus; use somente se quiser reiniciar o ambiente do zero. Logs têm rotação e o Prometheus retém sete dias.

## Decisões e limites

A API usa `net/http` e a biblioteca oficial do Prometheus. O binário estático roda sem root em imagem `scratch`, com filesystem somente leitura e sem capabilities. O servidor tem timeouts e encerramento gracioso. As rotas desconhecidas são agrupadas em `unmatched` para limitar cardinalidade. As imagens têm versões explícitas; os tags não são travados por digest e não há atualização automática. Os pacotes Docker usam a versão disponível no repositório oficial na primeira instalação.

A entrega é um ambiente de demonstração em um único host: sem TLS, alta disponibilidade ou backup automatizado. Os bind mounts usam rótulos SELinux (`Z`) para o Rocky Linux. Em produção, a evolução incluiria TLS, limites de recursos dimensionados com carga, política de atualização, backups, alertas enviados e teste sintético do acesso via NGINX.

## Documentação

- [Guia técnico para entrevista](docs/GUIA-ENTREVISTA.md)
- [Roteiro da demonstração](docs/DEMONSTRACAO.md)
- [Mapa dos requisitos](docs/REQUISITOS.md)
- [Registro de validação](docs/VALIDACAO.md)

Referências: [Docker no Ubuntu](https://docs.docker.com/engine/install/ubuntu/), [Ansible Compose V2](https://docs.ansible.com/projects/ansible/latest/collections/community/docker/docker_compose_v2_module.html), [instrumentação Go](https://prometheus.io/docs/tutorials/instrumenting_http_server_in_go/) e [provisionamento Grafana](https://grafana.com/docs/grafana/latest/administration/provisioning/).
