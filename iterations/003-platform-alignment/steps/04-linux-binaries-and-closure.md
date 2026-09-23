# Etapa: Binários Linux e fechamento

Status: `todo`

## Objetivo

Gerar `platformctl` para `linux/amd64` e `linux/arm64` dentro do container
lab, para levar à VM do Packer (fase 3, no `platform-modules`), e fechar a
iteração.

## Escopo

- build cruzado via `GOOS`/`GOARCH` no container lab, nunca no host
- checksums SHA-256 registrados como evidência
- fechamento da iteração, da issue #27 e atualização da tabela do programa

## Fora de Escopo

- empacotamento de release ou publicação de binários
- execução de qualquer coisa na VM do Packer (fase 3)

## Critérios de Aceite

- binários gerados fora do checkout, com checksums registrados
- `README.md` desta iteração com o resultado final

## Validação

```bash
cd container && ./lab.sh up && ./lab.sh status
```

## Resultado

Preencher ao concluir a etapa.

## Evidências

Preencher ao concluir a etapa.

## Pendências

Preencher ao concluir a etapa.
