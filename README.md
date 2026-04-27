# Spazio Backend

Backend API para Spazio, una plataforma inmobiliaria orientada a gestion de propiedades, clientes, operaciones y pagos.

## Contexto del sistema

Spazio centraliza procesos del negocio inmobiliario:

- Publicacion y administracion de propiedades
- Gestion de usuarios con roles y permisos
- Seguimiento comercial (inquiries, visitas, asignaciones)
- Operaciones de cierre (transacciones, contratos, pagos)
- Historial y auditoria de estados

## Stack tecnico

- Lenguaje: Go
- Framework HTTP: Gin
- Base de datos: PostgreSQL
- Driver DB: pgx v5 (pool de conexiones)
- Migraciones: migration/sql
- Consultas SQL: sqlc/queries (fuente de verdad para SQL)

## Arquitectura (Vertical Slicing)

El proyecto sigue arquitectura por modulos de dominio. Cada modulo contiene su propio flujo de entrega completo:

- handler: capa HTTP y registro de rutas
- service: logica de negocio
- repository: acceso a datos
- model: contratos (interfaces), DTOs y estructuras del modulo
- module: composition root del modulo (wiring manual)

### Flujo de una request

1. Handler recibe request y valida entrada
2. Service aplica reglas del caso de uso
3. Repository ejecuta operaciones de persistencia
4. Handler responde JSON

### Estructura actual

internal/

- config/
- db/
- middleware/
- modules/
  - properties/
    - handler.go
    - service.go
    - repository.go
    - model.go
    - module.go
- shared/
- sqlcgen/

## Resumen de entidades de dominio

A nivel de negocio, el modelo se organiza en 10 grupos:

1. Users, Security y RBAC

- Users, Roles, Permissions, RolePermissions, UserStatus

2. Properties (core)

- Properties, PropertyTypes, Modalities, PropertyStatus

3. Specialization

- ResidentialProperties, CommercialProperties, Orientations

4. Location y Geography

- Locations, Countries, States, Zones

5. Financials

- Prices, RentPeriods

6. Multimedia y Analytics

- PropertyPhotos, PropertyEvents

7. Services y Clauses

- Services, ServiceCategories, PropertyServices, Clauses, ClauseValueTypes, PropertyClauses

8. CRM y Logistics

- Inquiries, FollowUpStatus, PropertyAgents, AgentSchedules, PropertyExceptions, Visits, VisitStatus

9. Operations, Contracts y Payments

- Transactions, TransactionStatus, Contracts, ContractStatus, Payments, PaymentGateways, PaymentMethods, PaymentStatus

10. Audit y History

- PropertyStatusHistory, ContractStatusHistory, VisitStatusHistory, TransactionStatusHistory

## Convenciones clave

- Respuestas HTTP en JSON
- Wiring manual (sin framework de DI)
- Interfaces por modulo para mantener bajo acoplamiento
- Evitar SQL embebido en handlers o services

## Ejecucion local (referencia)

Variables de entorno minimas:

- APP_PORT
- DATABASE_URL

Comando tipico:

- go run ./cmd/api
