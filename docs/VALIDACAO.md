# Registro de validação

## Estado atual

- Código, Dockerfile, Compose, dashboard e automação implementados.
- Dependências Go resolvidas e `go.sum` gerado.
- Testes Go e integração chegaram a ser iniciados localmente, mas a execução foi interrompida. Não há resultado conclusivo registrado desses testes.
- Instalação Ansible adaptada para Rocky Linux 9/10; execução completa ainda pendente do acesso SSH à VM.
- Rede da VM responde a ping; a conexão SSH mais recente foi recusada antes da autenticação.
- Ubuntu no WSL não foi instalado. Apenas uma imagem de instalação foi baixada fora do repositório antes da escolha do Rocky Linux.

## Verificações restantes

1. Confirmar versão do Rocky Linux e compatibilidade do Python.
2. Executar `go vet ./...` e `go test -race -cover ./...`.
3. Executar o playbook completo e repetir para observar idempotência.
4. Validar NGINX com `nginx -t` e Prometheus com `promtool check config`.
5. Executar `scripts/smoke.py` com a stack ativa.
6. Confirmar acesso pelo Windows, dashboard, geração de tráfego e recuperação após parada da API.
7. Publicar no repositório indicado e acompanhar o CI.

Este registro deve ser atualizado com os resultados efetivos; configuração preparada ou syntax-check não equivale a provisionamento validado.
