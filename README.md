# Projeto Korp

API em Go com NGINX como proxy reverso e monitoramento com Prometheus e Grafana.

```text
Cliente → NGINX (:80) → http-server-projeto-korp (:8080)
                              ↑ coleta /metrics
                          Prometheus ← Grafana
```

Os containers usam a rede bridge `projeto-korp-network`. A API é acessada pelo NGINX na porta 80. Grafana publica a porta 3000 para acesso pela rede; Prometheus fica em `127.0.0.1:9090`.

## Executar com Docker Compose

Requer Docker Engine com o plugin Compose e a porta 80 livre.

```bash
cp .env.example .env
# Edite .env e defina GRAFANA_ADMIN_PASSWORD antes de iniciar.
docker network create --driver bridge projeto-korp-network
docker compose up --build -d --wait
curl http://localhost:80/projeto-korp
```

Se a rede já existir, reutilize-a. Sua criação fica fora do Compose porque também é uma tarefa explícita do Ansible.

Exemplo de resposta:

```json
{"nome":"Projeto Korp","horario":"2026-09-05T20:30:00Z"}
```

O horário é calculado em UTC a cada requisição. A aplicação aceita `GET /projeto-korp`, retorna 405 para outros métodos nessa rota e 404 para caminhos desconhecidos.

## Provisionar com Ansible

O playbook tem tarefas para Rocky Linux 9/10 e Ubuntu 24.04. Depois de preparar o controlador e o inventário conforme [ansible/README.md](ansible/README.md), execute:

```bash
ansible-playbook -i ansible/inventory.ini ansible/playbook.yml --ask-become-pass
```

Ele instala Docker, cria a rede, copia o projeto para `/opt/projeto-korp`, constrói a imagem e inicia os serviços. Ao final, verifica API, coleta do Prometheus e dashboard do Grafana, e exibe a resposta JSON no console.

## Monitoramento

Abra `http://IP_DO_SERVIDOR:3000` (ou `http://localhost:3000` no próprio servidor) e entre como `admin` com a senha definida no `.env`. O datasource e o dashboard **Projeto Korp — Serviço HTTP** são carregados dos arquivos em `monitoring/grafana`.

| Métrica | Uso |
| --- | --- |
| `up` | Sucesso ou falha da coleta pelo Prometheus |
| `http_requests_total` | Requisições por método, rota e status |
| `http_request_duration_seconds` | Latência do handler |

As consultas a `/healthz` e `/metrics` não entram no contador de tráfego. O NGINX bloqueia o acesso público a `/metrics`; o Prometheus coleta diretamente pela rede Docker. `up` não verifica o caminho completo pelo proxy.

Para movimentar os gráficos, execute `sh scripts/load.sh`. A regra `KorpServiceDown` entra em firing após 30 segundos de falha de coleta. Não há envio de notificações configurado.

## Testes

```bash
go vet ./...
go test -race -cover ./...
```

Com a stack ativa e `GRAFANA_ADMIN_PASSWORD` exportada no terminal:

```bash
python3 scripts/smoke.py
```

O [workflow de CI](.github/workflows/ci.yml) verifica formatação, testes Go, sintaxe Ansible, build e integração dos containers. A execução completa do playbook no Rocky Linux ainda está pendente; o teste de sintaxe não cobre essa instalação.

## Operação e escolhas

Use `docker compose ps` para consultar os serviços e `docker compose logs --tail=100` para diagnóstico. `docker compose down` preserva os volumes; `docker compose down -v` apaga os dados do Grafana e Prometheus.

A API usa `net/http`, com timeouts e encerramento gracioso. O build em dois estágios gera um binário estático que roda sem root em `scratch`. As configurações são montadas como somente leitura; os mounts usam `Z` para rotulagem SELinux. Os logs têm rotação e o Prometheus retém sete dias de dados.

O ambiente usa um único host, sem TLS ou alta disponibilidade. As imagens têm tags de versão explícitas, mas não estão fixadas por digest.
