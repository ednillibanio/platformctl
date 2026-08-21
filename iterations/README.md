# Iterations

Este diretório organiza o roteiro operacional do projeto em iterações e
etapas.

Cada iteração é um plano executável. Ela pode ser planejada e refinada sem
executar nenhuma alteração técnica. A execução só acontece quando houver
confirmação explícita.

## Objetivo

- manter um roteiro claro de melhorias e correções
- agrupar atividades por finalidade
- registrar progresso por etapa
- preservar evidências de validação
- evitar execução automática sem decisão do operador

## Fluxo

1. Propor uma iteração com objetivo, escopo e etapas.
2. Aguardar confirmação explícita para criar os arquivos da iteração.
3. Criar uma branch própria a partir de `lab` (nunca commitar direto em `lab`
   ou `main`). `main` é a branch principal; `lab` é a branch de integração e
   validação antes da promoção.
4. Executar uma etapa por vez, somente após confirmação.
5. Registrar resultado, validação e pendências da etapa.
6. Fechar a etapa.
7. Avançar para a próxima etapa somente após nova confirmação.
8. Fechar a iteração quando todas as etapas estiverem `done` ou `skipped`.
9. Abrir PR da branch da iteração para `lab`, revisar e integrar.
10. Depois da validação integrada em `lab`, promover `lab` para `main` por um
    PR revisado separado.

## Status

- `todo`: ainda não iniciada.
- `doing`: etapa ou iteração em andamento.
- `blocked`: depende de decisão, credencial, ambiente ou outro recurso.
- `done`: concluída, validada e registrada.
- `skipped`: descartada com justificativa registrada.

## Regra de Execução

Nenhuma iteração ou etapa deve ser executada automaticamente.

A confirmação explícita é obrigatória para:

- iniciar uma iteração
- iniciar uma etapa
- aplicar correções
- alterar arquivos fora desta área de planejamento
- fechar uma etapa
- fechar uma iteração
- commitar, empurrar (push) ou abrir PR

## Regra de Módulos

Iterações específicas por módulo ou pasta devem ser criadas sob demanda.
Não crie uma iteração genérica sem demanda explícita.

## Evidência

Toda etapa fechada deve registrar:

- o que foi feito
- quais arquivos foram alterados
- quais comandos de validação foram executados
- resultado da validação
- pendências ou riscos restantes

## Issue

Cada iteração tem exatamente uma issue no GitHub (template "Iteração",
label `iteration`), nunca uma issue por etapa. A issue espelha a tabela de
etapas do `README.md` da iteração e é atribuída ao operador.

## Documentação

Os arquivos nesta pasta são documentação operacional interna do projeto.

Use português nesta área. Mantenha scripts, comentários internos de código,
CLI help, logs e mensagens de erro em inglês, conforme o padrão do
repositório (ver `CLAUDE.md`).
