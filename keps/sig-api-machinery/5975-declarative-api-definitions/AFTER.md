# After: Adding a New API Resource (With DV + Strategy DSL)

This shows the same `Widget` resource as BEFORE.md, but using declarative
validation (DV) and the strategy DSL. Only files that **differ** from the
before state are shown — type definitions, register.go, REST install wiring,
and printer registration are unchanged.

---

## 1. Type Definitions — now carry validation tags

### `staging/src/k8s.io/api/widgets/v1/types.go` (same size, tags added)

```go
type WidgetSpec struct {
    // Size is the desired size of the widget.
    // +optional
    // +k8s:minimum=1
    // +k8s:maximum=100
    Size *int32 `json:"size,omitempty" protobuf:"..."`

    // Color is the widget color.
    // +optional
    // +k8s:enum=red;green;blue
    Color string `json:"color,omitempty" protobuf:"..."`
}
```

The validation rules are declared on the fields. No separate validation
file is needed.

---

## 2. Validation — eliminated entirely

With 100% declarative validation, there is no `validation.go` file at all.
All constraints are expressed via DV tags on the type fields (`+k8s:minimum`,
`+k8s:maximum`, `+k8s:enum`, `+k8s:required`, etc.). The generated
declarative validation functions are registered in the scheme and invoked
automatically by the strategy.

For resources with complex cross-field validation that cannot be expressed
declaratively, a `validation.go` file would still exist but contain only those
non-declarative rules:

```go
func ValidateWidget(widget *widgets.Widget) field.ErrorList {
    // Only non-declarative rules go here.
    // For this simple resource, there are none.
    return nil
}
```

And the strategy would reference it:

```go
Validate: func(ctx context.Context, obj runtime.Object) field.ErrorList {
    return validation.ValidateWidget(obj.(*widgets.Widget))
},
```

But for Widget, neither the file nor the config field is needed.

---

## 3. Strategy — 8 lines of config

### `pkg/registry/widgets/widget/strategy.go`

```go
package widget

import (
    "k8s.io/apiserver/pkg/registry/rest/strategy"
    "k8s.io/kubernetes/pkg/api/legacyscheme"
    "k8s.io/kubernetes/pkg/apis/widgets"
)

var Strategy = strategy.New(strategy.Config{
    Object:      &widgets.Widget{},
    Scheme:      legacyscheme.Scheme,
    Namespaced:  true,
})
```

Declarative validation is invoked automatically. When no hand-written
`Validate`/`ValidateUpdate` functions are provided, declarative validation
runs as the sole validation path.

The DSL handles automatically:
- Status clearing on create (detected via `StatusAccessor` on the type)
- Status clearing on update (copies old status to prevent modification)
- Generation = 1 on create
- Generation bump when Spec changes (detected via `SpecAccessor` on the type)
- Status substrategy (`Strategy.StatusStrategy`)
- Declarative validation
- `GetResetFields`, `DefaultGarbageCollectionPolicy`, `AllowCreateOnUpdate`,
  `AllowUnconditionalUpdate`, `Canonicalize`, `WarningsOnCreate/Update` (all defaults)

---

## 4. Storage — 30 lines

### `pkg/registry/widgets/widget/storage/storage.go`

```go
package storage

import (
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apiserver/pkg/registry/generic"
    genericregistry "k8s.io/apiserver/pkg/registry/generic/registry"
    "k8s.io/kubernetes/pkg/apis/widgets"
    "k8s.io/kubernetes/pkg/printers"
    printersinternal "k8s.io/kubernetes/pkg/printers/internalversion"
    printerstorage "k8s.io/kubernetes/pkg/printers/storage"
    widgetstrategy "k8s.io/kubernetes/pkg/registry/widgets/widget"
)

type REST struct{ *genericregistry.Store }
type StatusREST struct{ store *genericregistry.Store }

func NewREST(optsGetter generic.RESTOptionsGetter) (*REST, *StatusREST, error) {
    store := &genericregistry.Store{
        NewFunc:                   func() runtime.Object { return &widgets.Widget{} },
        NewListFunc:               func() runtime.Object { return &widgets.WidgetList{} },
        DefaultQualifiedResource:  widgets.Resource("widgets"),
        SingularQualifiedResource: widgets.Resource("widget"),
        CreateStrategy:            widgetstrategy.Strategy,
        UpdateStrategy:            widgetstrategy.Strategy,
        DeleteStrategy:            widgetstrategy.Strategy,
        TableConvertor: printerstorage.TableConvertor{
            TableGenerator: printers.NewTableGenerator().With(printersinternal.AddHandlers),
        },
    }
    if err := store.CompleteWithOptions(&generic.StoreOptions{RESTOptions: optsGetter}); err != nil {
        return nil, nil, err
    }
    statusStore := *store
    statusStore.UpdateStrategy = widgetstrategy.Strategy.StatusStrategy
    statusStore.ResetFieldsStrategy = widgetstrategy.Strategy.StatusStrategy
    return &REST{store}, &StatusREST{store: &statusStore}, nil
}

// StatusREST methods (same as before — this boilerplate is not yet addressed)
func (r *StatusREST) New() runtime.Object { return &widgets.Widget{} }
func (r *StatusREST) Destroy()            {}
// ... Get, Update, GetResetFields ...
```

---

## Summary: What Changed

| File | Before | After | Reduction |
|------|--------|-------|-----------|
| types.go (versioned) | 30 lines | 30 lines (+ DV tags) | 0 (tags replace validation.go) |
| validation.go | 50 lines | 0 lines (eliminated) | **-50 lines** |
| **strategy.go** | **100 lines** | **8 lines** | **-92 lines** |
| storage.go | 70 lines | 45 lines | **-25 lines** |
| REST install wiring | 15 lines | 15 lines | 0 |
| Printer registration | 30 lines | 30 lines | 0 |
| **Total hand-written** | **~330 lines** | **~160 lines** | **-170 lines (52%)** |

The strategy file shrinks from 100 lines (struct + 12 methods + status
substrategy) to 8 lines (three config fields). The validation file is
eliminated entirely — its constraints now live as DV tags on the type fields.

The remaining hand-written code is genuinely resource-specific:
- Type definitions with DV tags (the API surface itself)
- Storage wiring (resource names, printer setup)
- Printer column definitions and extraction logic
- REST install registration
