# Registro de validação

## Estado atual

- Código, Dockerfile, Compose, dashboard e automação implementados.
- Dependências Go resolvidas e `go.sum` gerado.
- GitHub Actions: formatação, `go vet` e `go test -race -cover` passaram; cobertura registrada de 56,2% dos statements.
- Verificação de sintaxe Ansible passou.
- Stack construída e iniciada no runner Linux; validações NGINX e Prometheus passaram.
- Teste integrado passou: contrato JSON, horário UTC dinâmico, bloqueio público de métricas, scrape e dashboard provisionado.
- A primeira execução do CI falhou na checagem final de portas, depois dos testes integrados. A checagem foi alterada para verificar `HostConfig.PortBindings` diretamente no Docker; a nova execução deve confirmar o resultado global.
- Instalação Ansible adaptada para Rocky Linux 9/10; execução completa ainda pendente do acesso SSH à VM.
- Rede da VM responde a ping; a conexão SSH mais recente foi recusada antes da autenticação.
- Ubuntu no WSL não foi instalado. Apenas uma imagem de instalação foi baixada fora do repositório antes da escolha do Rocky Linux.

## Verificações restantes

1. Confirmar versão do Rocky Linux e compatibilidade do Python.
2. Confirmar o resultado global do CI após a correção da checagem de portas.
3. Executar o playbook completo e repetir para observar idempotência.
4. Repetir as verificações NGINX/Prometheus no Rocky Linux de destino.
5. Executar `scripts/smoke.py` na VM, além do resultado já obtido no CI.
6. Confirmar acesso pelo Windows, dashboard, geração de tráfego e recuperação após parada da API.
7. Publicar no repositório indicado e acompanhar o CI.

Este registro deve ser atualizado com os resultados efetivos; configuração preparada ou syntax-check não equivale a provisionamento validado.
