## Plan de endpoints — Edición de propiedad

### Contexto y decisiones confirmadas

- `modality_id` y `category` **no son editables** después de crear la propiedad
- El **frontend detecta cambios** y solo llama los endpoints necesarios
- El **backend compara antes de escribir** — sin cambios reales, no hay escritura
- Todos los endpoints nuevos viven en el módulo `properties`

---

### Endpoints reutilizables sin modificación

| Endpoint                                       | Uso en el formulario                            |
| ---------------------------------------------- | ----------------------------------------------- |
| `GET /api/v1/catalogs/modalities`              | Select de modalidad (solo lectura, no editable) |
| `GET /api/v1/catalogs/property-types`          | Select de tipo de propiedad                     |
| `GET /api/v1/catalogs/rent-periods`            | Opciones de período en precios                  |
| `GET /api/v1/catalogs/orientations`            | Select de orientación en subtipo residencial    |
| `GET /api/v1/clauses`                          | Catálogo de cláusulas disponibles               |
| `GET /api/v1/services`                         | Catálogo de servicios disponibles               |
| `GET /api/v1/locations/countries`              | Selector de país                                |
| `GET /api/v1/locations/states`                 | Selector de estado                              |
| `GET /api/v1/locations/cities`                 | Selector de ciudad                              |
| `POST /api/v1/uploads/properties/:uuid/photos` | Subir fotos nuevas durante edición              |

---

### Endpoints nuevos — módulo `properties`

---

#### `GET /api/v1/properties/:uuid`

Carga datos base, subtipo (residential o commercial según `category`) y ubicación.

**Response `200`**

```json
{
  "data": {
    "property_uuid": "123e4567-...",
    "owner_id": 1,
    "category": "residential",
    "title": "Casa en Xalapa",
    "description": "Espaciosa propiedad cerca del centro",
    "property_type_id": 1,
    "modality_id": 1,
    "lot_area": 200.0,
    "is_featured": false,
    "residential": {
      "bedrooms": 3,
      "bathrooms": 2,
      "beds": 4,
      "floors": 2,
      "parking_spots": 1,
      "built_area": 120.0,
      "construction_year": 2010,
      "orientation_id": 2,
      "is_furnished": false
    },
    "commercial": null,
    "location": {
      "city_id": 1,
      "neighborhood": "Centro",
      "street": "Av. Principal",
      "exterior_number": "45",
      "interior_number": null,
      "postal_code": "91000",
      "latitude": 19.5438,
      "longitude": -96.9102,
      "is_public_address": true
    }
  }
}
```

---

#### `PATCH /api/v1/properties/:uuid`

Edita datos base, subtipo y ubicación en una sola transacción.

**Campos no aceptados en el payload:** `category`, `modality_id` — si se envían, `400`.

**Lógica de persistencia:**

- El backend lee el estado actual y compara campo a campo
- Si ningún campo difiere, retorna `204` sin escribir nada
- Si hay diferencias, actualiza `properties` y la tabla de especialización correspondiente con `updated_at = NOW()`

**Request**

```json
{
  "title": "Casa remodelada en Xalapa",
  "description": "...",
  "property_type_id": 1,
  "lot_area": 210.0,
  "is_featured": true,
  "residential": {
    "bedrooms": 4,
    "bathrooms": 3,
    "beds": 5,
    "floors": 2,
    "parking_spots": 2,
    "built_area": 135.0,
    "construction_year": 2010,
    "orientation_id": 2,
    "is_furnished": true
  },
  "location": {
    "city_id": 1,
    "neighborhood": "Centro",
    "street": "Av. Principal",
    "exterior_number": "45",
    "interior_number": null,
    "postal_code": "91000",
    "latitude": 19.5438,
    "longitude": -96.9102,
    "is_public_address": true
  }
}
```

**Responses**

- `204` — Sin cambios detectados o cambios persistidos correctamente
- `400` — Payload inválido o campos no editables presentes
- `404` — Propiedad no encontrada

---

#### `GET /api/v1/properties/:uuid/prices`

Devuelve únicamente los registros activos (`is_current = true`). El historial no se expone.

**Response `200`**

```json
{
  "data": {
    "sale_price": {
      "sale_price": 1500000.0,
      "currency": "MXN",
      "is_negotiable": true
    },
    "rent_prices": [
      {
        "period_id": 3,
        "rent_price": 8000.0,
        "deposit": 16000.0,
        "currency": "MXN",
        "is_negotiable": false
      }
    ]
  }
}
```

Campos nulos según modalidad: si `modality_id = 1` (venta), `rent_prices` viene como `[]`. Si `modality_id = 2` (renta), `sale_price` viene como `null`.

---

#### `PUT /api/v1/properties/:uuid/prices`

Reemplaza los precios activos respetando el historial. El backend lee `modality_id` de `properties` para validar el payload.

**Lógica de persistencia por cada precio:**

1. Compara el precio entrante contra el registro activo actual
2. Si no hay diferencia en ningún campo, no escribe nada para ese precio
3. Si hay diferencia: `valid_until = NOW()`, `is_current = false` en el registro activo; inserta nuevo con `valid_from = NOW()`, `is_current = true`, `changed_by_user_id = owner_id`

**Garantía:** nunca existirán dos registros con `is_current = true` para el mismo inmueble y mismo tipo de precio.

**Request**

```json
{
  "sale_price": {
    "sale_price": 1700000.0,
    "currency": "MXN",
    "is_negotiable": false
  },
  "rent_prices": [
    {
      "period_id": 3,
      "rent_price": 9000.0,
      "deposit": 18000.0,
      "currency": "MXN",
      "is_negotiable": false
    }
  ]
}
```

**Responses**

- `204` — Sin cambios o cambios persistidos correctamente
- `400` — Payload no corresponde a la modalidad de la propiedad
- `404` — Propiedad no encontrada

---

#### `GET /api/v1/properties/:uuid/photos`

Metadatos de todas las fotos vinculadas a la propiedad.

**Response `200`**

```json
{
  "data": [
    {
      "photo_id": 1,
      "storage_key": "properties/uuid/foto1.jpg",
      "mime_type": "image/jpeg",
      "sort_order": 1,
      "is_cover": true,
      "label": "Fachada",
      "alt_text": "Vista frontal de la casa"
    }
  ]
}
```

---

#### `PUT /api/v1/properties/:uuid/photos`

Sincroniza metadatos de fotos. Las fotos ya existen en storage.

**Lógica de persistencia:**

- Elimina registros en `property_photos` cuyos `photo_id` no estén en el payload
- Actualiza `sort_order`, `is_cover`, `label`, `alt_text` de los que sí están
- Si `is_cover` cambió, actualiza `cover_photo_url` en `properties`
- El backend compara antes de escribir cada campo

**Regla de integridad:** exactamente un elemento debe tener `is_cover = true`. Si ninguno o más de uno, `400`.

**Request**

```json
{
  "photos": [
    {
      "photo_id": 1,
      "sort_order": 2,
      "is_cover": false,
      "label": "Fachada",
      "alt_text": "Vista frontal"
    },
    {
      "photo_id": 3,
      "sort_order": 1,
      "is_cover": true,
      "label": "Sala principal",
      "alt_text": "Sala amplia con luz natural"
    }
  ]
}
```

**Responses**

- `204` — Sin cambios o cambios persistidos correctamente
- `400` — Ninguna o más de una foto marcada como portada
- `404` — Propiedad no encontrada

---

#### `GET /api/v1/properties/:uuid/services`

IDs de servicios actualmente vinculados a la propiedad.

**Response `200`**

```json
{
  "data": {
    "service_ids": [1, 3, 7]
  }
}
```

---

#### `PUT /api/v1/properties/:uuid/services`

Sincronización completa. El backend hace diff: elimina los que no están en el payload, inserta los nuevos con `ON CONFLICT DO NOTHING`.

**Request**

```json
{
  "service_ids": [1, 5, 9]
}
```

**Responses**

- `204` — Sin cambios o cambios persistidos correctamente
- `400` — Algún `service_id` inválido
- `404` — Propiedad no encontrada

---

#### `GET /api/v1/properties/:uuid/clauses`

Cláusulas actualmente vinculadas con sus valores.

**Response `200`**

```json
{
  "data": [
    {
      "clause_id": 1,
      "boolean_value": true,
      "integer_value": null,
      "min_value": null,
      "max_value": null
    }
  ]
}
```

---

#### `PUT /api/v1/properties/:uuid/clauses`

Elimina todas las cláusulas actuales de la propiedad e inserta el nuevo conjunto completo. Las validaciones de `value_type` aplican igual que en el registro inicial.

**Request**

```json
{
  "clauses": [
    {
      "clause_id": 1,
      "boolean_value": true,
      "integer_value": null,
      "min_value": null,
      "max_value": null
    },
    {
      "clause_id": 4,
      "boolean_value": null,
      "integer_value": null,
      "min_value": 1.0,
      "max_value": 3.0
    }
  ]
}
```

**Responses**

- `204` — Cambios persistidos correctamente
- `400` — Payload de cláusula inválido
- `404` — Propiedad no encontrada

---

### Resumen completo

| Método  | Ruta                                | Propósito                               |
| ------- | ----------------------------------- | --------------------------------------- |
| `GET`   | `/api/v1/properties/:uuid`          | Datos base, subtipo y ubicación         |
| `PATCH` | `/api/v1/properties/:uuid`          | Editar datos base, subtipo y ubicación  |
| `GET`   | `/api/v1/properties/:uuid/prices`   | Precios activos                         |
| `PUT`   | `/api/v1/properties/:uuid/prices`   | Reemplazar precios respetando historial |
| `GET`   | `/api/v1/properties/:uuid/photos`   | Metadatos de fotos                      |
| `PUT`   | `/api/v1/properties/:uuid/photos`   | Sincronizar metadatos y portada         |
| `GET`   | `/api/v1/properties/:uuid/services` | Servicios vinculados                    |
| `PUT`   | `/api/v1/properties/:uuid/services` | Sincronizar servicios                   |
| `GET`   | `/api/v1/properties/:uuid/clauses`  | Cláusulas vinculadas                    |
| `PUT`   | `/api/v1/properties/:uuid/clauses`  | Sincronizar cláusulas                   |

¿Arrancamos con la implementación? Si es así, ¿por cuál endpoint quieres empezar?
