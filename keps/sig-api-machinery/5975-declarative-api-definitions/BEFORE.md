# Before: Adding a New API Resource (Today)

This shows the complete hand-written code required to add a namespaced resource
`Widget` with Spec and Status to a `widgets.example.io` API group today,
**without** declarative validation or the strategy DSL.

Only hand-written files are shown. Generated files (deepcopy, conversion,
openapi, protobuf, apply configurations) are omitted.

---

## 1. Type Definitions

### `pkg/apis/widgets/types.go` (~30 lines for the resource)

```go
type Widget struct {
    metav1.TypeMeta
    metav1.ObjectMeta
    Spec   WidgetSpec
    Status WidgetStatus
}

type WidgetSpec struct {
    // Size is the desired size of the widget. Must be between 1 and 100.
    // +optional
    Size *int32
    // Color is the widget color.
    // +optional
    Color string
}

type WidgetStatus struct {
    // Phase is the current lifecycle phase.
    Phase string
    // Ready indicates the widget is operational.
    Ready bool
}

type WidgetList struct {
    metav1.TypeMeta
    metav1.ListMeta
    Items []Widget
}
```

### `staging/src/k8s.io/api/widgets/v1/types.go` (~30 lines)

Same structs with JSON and protobuf tags:

```go
type Widget struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty" protobuf:"..."`
    Spec   WidgetSpec   `json:"spec,omitempty" protobuf:"..."`
    Status WidgetStatus `json:"status,omitempty" protobuf:"..."`
}

type WidgetSpec struct {
    // +optional
    Size *int32 `json:"size,omitempty" protobuf:"..."`
    // +optional
    Color string `json:"color,omitempty" protobuf:"..."`
}

// ... WidgetStatus, WidgetList with JSON tags ...
```

### `register.go` edits (2 lines each, 2 files)

```go
// pkg/apis/widgets/register.go — add to addKnownTypes:
scheme.AddKnownTypes(SchemeGroupVersion, &Widget{}, &WidgetList{})

// staging/src/k8s.io/api/widgets/v1/register.go — add to AddKnownTypes:
scheme.AddKnownTypes(SchemeGroupVersion, &Widget{}, &WidgetList{})
```

---

## 2. Validation (~50 lines)

### `pkg/apis/widgets/validation/validation.go`

```go
func ValidateWidget(widget *widgets.Widget) field.ErrorList {
    allErrs := apivalidation.ValidateObjectMeta(
        &widget.ObjectMeta, true,
        apivalidation.NameIsDNSSubdomain, field.NewPath("metadata"))
    allErrs = append(allErrs, validateWidgetSpec(&widget.Spec, field.NewPath("spec"))...)
    return allErrs
}

func ValidateWidgetUpdate(new, old *widgets.Widget) field.ErrorList {
    allErrs := apivalidation.ValidateObjectMetaUpdate(&new.ObjectMeta, &old.ObjectMeta, field.NewPath("metadata"))
    allErrs = append(allErrs, validateWidgetSpec(&new.Spec, field.NewPath("spec"))...)
    return allErrs
}

func validateWidgetSpec(spec *widgets.WidgetSpec, fldPath *field.Path) field.ErrorList {
    var allErrs field.ErrorList
    if spec.Size != nil {
        if *spec.Size < 1 || *spec.Size > 100 {
            allErrs = append(allErrs, field.Invalid(
                fldPath.Child("size"), *spec.Size, "must be between 1 and 100"))
        }
    }
    if spec.Color != "" {
        validColors := sets.New("red", "green", "blue")
        if !validColors.Has(spec.Color) {
            allErrs = append(allErrs, field.NotSupported(
                fldPath.Child("color"), spec.Color, sets.List(validColors)))
        }
    }
    return allErrs
}
```

---

## 3. Strategy (~100 lines)

### `pkg/registry/widgets/widget/strategy.go`

```go
type widgetStrategy struct {
    runtime.ObjectTyper
    names.NameGenerator
}

var Strategy = widgetStrategy{legacyscheme.Scheme, names.SimpleNameGenerator}

func (widgetStrategy) NamespaceScoped() bool { return true }

func (widgetStrategy) DefaultGarbageCollectionPolicy(
    ctx context.Context) rest.GarbageCollectionPolicy {
    return rest.DeleteDependents
}

func (widgetStrategy) PrepareForCreate(ctx context.Context, obj runtime.Object) {
    widget := obj.(*widgets.Widget)
    widget.Status = widgets.WidgetStatus{}
    widget.Generation = 1
}

func (widgetStrategy) Validate(
    ctx context.Context, obj runtime.Object) field.ErrorList {
    return validation.ValidateWidget(obj.(*widgets.Widget))
}

func (widgetStrategy) WarningsOnCreate(
    ctx context.Context, obj runtime.Object) []string {
    return nil
}

func (widgetStrategy) Canonicalize(obj runtime.Object) {}

func (widgetStrategy) AllowCreateOnUpdate() bool { return false }

func (widgetStrategy) PrepareForUpdate(
    ctx context.Context, obj, old runtime.Object) {
    newWidget := obj.(*widgets.Widget)
    oldWidget := old.(*widgets.Widget)
    newWidget.Status = oldWidget.Status
    if !apiequality.Semantic.DeepEqual(newWidget.Spec, oldWidget.Spec) {
        newWidget.Generation = oldWidget.Generation + 1
    }
}

func (widgetStrategy) ValidateUpdate(
    ctx context.Context, obj, old runtime.Object) field.ErrorList {
    return validation.ValidateWidgetUpdate(
        obj.(*widgets.Widget), old.(*widgets.Widget))
}

func (widgetStrategy) WarningsOnUpdate(
    ctx context.Context, obj, old runtime.Object) []string {
    return nil
}

func (widgetStrategy) AllowUnconditionalUpdate() bool { return true }

func (widgetStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
    return map[fieldpath.APIVersion]*fieldpath.Set{}
}

// --- Status substrategy ---

type widgetStatusStrategy struct {
    widgetStrategy
}

var StatusStrategy = widgetStatusStrategy{Strategy}

func (widgetStatusStrategy) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
    return map[fieldpath.APIVersion]*fieldpath.Set{}
}

func (widgetStatusStrategy) PrepareForUpdate(
    ctx context.Context, obj, old runtime.Object) {
    newWidget := obj.(*widgets.Widget)
    oldWidget := old.(*widgets.Widget)
    newWidget.Spec = oldWidget.Spec
    newWidget.Labels = oldWidget.Labels
    newWidget.Annotations = oldWidget.Annotations
}

func (widgetStatusStrategy) ValidateUpdate(
    ctx context.Context, obj, old runtime.Object) field.ErrorList {
    return validation.ValidateWidgetUpdate(
        obj.(*widgets.Widget), old.(*widgets.Widget))
}

func (widgetStatusStrategy) WarningsOnUpdate(
    ctx context.Context, obj, old runtime.Object) []string {
    return nil
}
```

---

## 4. Storage (~70 lines)

### `pkg/registry/widgets/widget/storage/storage.go`

```go
type REST struct {
    *genericregistry.Store
}

type StatusREST struct {
    store *genericregistry.Store
}

func NewREST(optsGetter generic.RESTOptionsGetter) (
    *REST, *StatusREST, error) {
    store := &genericregistry.Store{
        NewFunc:     func() runtime.Object { return &widgets.Widget{} },
        NewListFunc: func() runtime.Object { return &widgets.WidgetList{} },
        DefaultQualifiedResource:  widgets.Resource("widgets"),
        SingularQualifiedResource: widgets.Resource("widget"),

        CreateStrategy:      widget.Strategy,
        UpdateStrategy:       widget.Strategy,
        DeleteStrategy:       widget.Strategy,
        ResetFieldsStrategy: widget.Strategy,

        TableConvertor: printerstorage.TableConvertor{
            TableGenerator: printers.NewTableGenerator().With(
                internalversion.AddHandlers),
        },
    }
    options := &generic.StoreOptions{RESTOptions: optsGetter}
    if err := store.CompleteWithOptions(options); err != nil {
        return nil, nil, err
    }

    statusStore := *store
    statusStore.UpdateStrategy = widget.StatusStrategy
    statusStore.ResetFieldsStrategy = widget.StatusStrategy

    return &REST{store}, &StatusREST{store: &statusStore}, nil
}

func (r *StatusREST) New() runtime.Object { return &widgets.Widget{} }
func (r *StatusREST) Destroy()            {}
func (r *StatusREST) Get(ctx context.Context, name string,
    options *metav1.GetOptions) (runtime.Object, error) {
    return r.store.Get(ctx, name, options)
}
func (r *StatusREST) Update(ctx context.Context, name string,
    objInfo rest.UpdatedObjectInfo, createValidation rest.ValidateObjectFunc,
    updateValidation rest.ValidateObjectUpdateFunc,
    forceAllowCreate bool,
    options *metav1.UpdateOptions) (runtime.Object, bool, error) {
    return r.store.Update(ctx, name, objInfo, createValidation,
        updateValidation, forceAllowCreate, options)
}
func (r *StatusREST) GetResetFields() map[fieldpath.APIVersion]*fieldpath.Set {
    return r.store.GetResetFields()
}
```

---

## 5. REST Install Wiring (~15 lines added)

### `pkg/registry/widgets/rest/storage_widgets.go` (edit)

```go
func (p StorageProvider) v1Storage(
    apiResourceConfigSource servicestorage.APIResourceConfigSource,
    restOptionsGetter generic.RESTOptionsGetter,
) (map[string]rest.Storage, error) {
    storage := map[string]rest.Storage{}

    if resource := "widgets"; apiResourceConfigSource.ResourceEnabled(
        widgetsv1.SchemeGroupVersion.WithResource(resource)) {
        widgetStorage, widgetStatusStorage, err :=
            widgetstorage.NewREST(restOptionsGetter)
        if err != nil {
            return storage, err
        }
        storage[resource] = widgetStorage
        storage[resource+"/status"] = widgetStatusStorage
    }

    return storage, nil
}
```

---

## 6. Printer Registration (~30 lines added)

### `pkg/printers/internalversion/printers.go` (edit)

```go
// Column definitions
var widgetColumnDefinitions = []metav1.TableColumnDefinition{
    {Name: "Name", Type: "string", Format: "name"},
    {Name: "Color", Type: "string"},
    {Name: "Size", Type: "integer"},
    {Name: "Ready", Type: "boolean"},
    {Name: "Age", Type: "string"},
}

// In AddHandlers():
h.TableHandler(widgetColumnDefinitions, printWidget)
h.TableHandler(widgetColumnDefinitions, printWidgetList)

// Print functions
func printWidget(obj *widgets.Widget,
    options printers.GenerateOptions) ([]metav1.TableRow, error) {
    row := metav1.TableRow{Object: runtime.RawExtension{Object: obj}}
    size := "<none>"
    if obj.Spec.Size != nil {
        size = fmt.Sprintf("%d", *obj.Spec.Size)
    }
    row.Cells = append(row.Cells,
        obj.Name, obj.Spec.Color, size,
        obj.Status.Ready, translateTimestampSince(obj.CreationTimestamp))
    return []metav1.TableRow{row}, nil
}

func printWidgetList(list *widgets.WidgetList,
    options printers.GenerateOptions) ([]metav1.TableRow, error) {
    rows := make([]metav1.TableRow, 0, len(list.Items))
    for i := range list.Items {
        r, err := printWidget(&list.Items[i], options)
        if err != nil {
            return nil, err
        }
        rows = append(rows, r...)
    }
    return rows, nil
}
```

---

## 7. Additional Per-Resource Wiring (~10 lines across 3 files)

### `pkg/kubeapiserver/default_storage_factory_builder.go` (edit, ~2 lines)

```go
// Storage version pinning
widgets.Resource("widgets").WithVersion("v1"),
```

### `plugin/pkg/auth/authorizer/rbac/bootstrappolicy/policy.go` (edit, ~5 lines)

```go
// Grant built-in roles access to the new resource
rbacv1helpers.NewRule("get","list","watch").Groups("widgets.example.io").
    Resources("widgets").RuleOrDie(),
```

### `hack/lib/init.sh` (edit, ~1 line)

```bash
# Add group version to the tracked list
widgets.example.io/v1
```

---

## Summary

| File | Lines | Content |
|------|-------|---------|
| Internal types.go | ~30 | Type definitions |
| Versioned types.go | ~30 | Type definitions + JSON tags |
| register.go (x2) | ~4 | AddKnownTypes calls |
| validation.go | ~50 | Hand-written validation |
| **strategy.go** | **~100** | **Strategy struct + 12 methods + status substrategy** |
| **storage.go** | **~70** | **Store config + StatusREST** |
| REST install wiring | ~15 | StorageProvider block |
| Printer registration | ~30 | Column defs + print functions |
| RBAC bootstrap policy | ~5 | Cluster role entries |
| Storage version pinning | ~2 | Resource→version mapping |
| hack/lib/init.sh | ~1 | Group version registration |
| **Total hand-written** | **~340** | |

The **bolded** files (strategy.go and storage.go) are almost entirely
boilerplate. Of the 170 lines, fewer than 10 carry resource-specific decisions
(namespaced=true, allowCreateOnUpdate=false, validation function names).
