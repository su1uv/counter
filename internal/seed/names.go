package seed

// Fixed word lists for "Joyería Aurora". Order matters: generation indexes
// into these slices, so reordering them changes every seeded dataset.

type site struct {
	Name, City, State, Tier string
}

// branchSites has 3 large, 7 medium and 5 small branches. The first three
// cover every tier, so small test configs still compare tiers.
var branchSites = []site{
	{"Centro", "Ciudad de México", "CDMX", "large"},
	{"Polanco", "Ciudad de México", "CDMX", "medium"},
	{"Coyoacán", "Ciudad de México", "CDMX", "small"},
	{"Santa Fe", "Ciudad de México", "CDMX", "large"},
	{"Del Valle", "Ciudad de México", "CDMX", "medium"},
	{"Chapultepec", "Guadalajara", "Jalisco", "medium"},
	{"Providencia", "Guadalajara", "Jalisco", "small"},
	{"Americana", "Guadalajara", "Jalisco", "medium"},
	{"San Pedro", "Monterrey", "Nuevo León", "large"},
	{"Cumbres", "Monterrey", "Nuevo León", "small"},
	{"Valle Oriente", "Monterrey", "Nuevo León", "medium"},
	{"Angelópolis", "Puebla", "Puebla", "medium"},
	{"La Paz", "Puebla", "Puebla", "small"},
	{"Juriquilla", "Querétaro", "Querétaro", "medium"},
	{"Centro Sur", "Querétaro", "Querétaro", "small"},
}

// categoryDef prices in whole pesos; zero means use the material's range.
type categoryDef struct {
	Code, Name, Singular string
	Styles               []string
	MinPesos, MaxPesos   int64
}

var categoryDefs = []categoryDef{
	{Code: "AN", Name: "Anillos", Singular: "Anillo", Styles: []string{"solitario", "de compromiso", "churumbela", "de sello", "trenzado", "con circonia"}},
	{Code: "AR", Name: "Aretes", Singular: "Aretes", Styles: []string{"de botón", "de arracada", "colgantes", "de perla", "de corazón"}},
	{Code: "CO", Name: "Collares", Singular: "Collar", Styles: []string{"de perlas", "con dije de corazón", "gargantilla", "de cruz", "riviera"}},
	{Code: "CA", Name: "Cadenas", Singular: "Cadena", Styles: []string{"cartier", "torzal", "veneciana", "barbada", "rolo"}},
	{Code: "PU", Name: "Pulseras", Singular: "Pulsera", Styles: []string{"esclava", "tenis", "de eslabones", "con dijes", "rígida"}},
	{Code: "DI", Name: "Dijes", Singular: "Dije", Styles: []string{"de corazón", "de la Virgen de Guadalupe", "de cruz", "de inicial", "de estrella"}},
	{Code: "RE", Name: "Relojes", Singular: "Reloj", Styles: []string{"clásico", "deportivo", "de vestir", "cronógrafo"}, MinPesos: 1500, MaxPesos: 25000},
	{Code: "JU", Name: "Juegos", Singular: "Juego", Styles: []string{"de collar y aretes", "nupcial", "de pulsera y aretes"}},
}

// materialDef prices in whole pesos.
type materialDef struct {
	Code, Label        string
	MinPesos, MaxPesos int64
}

var materials = []materialDef{
	{"PLATA", "plata .925", 300, 3000},
	{"ORO10", "oro 10k", 1500, 20000},
	{"ORO14", "oro 14k", 2500, 40000},
	{"ORO18", "oro 18k", 4000, 60000},
}

type supplierDef struct {
	Code, Name, City string
	LeadTimeDays     int
}

// supplierDefs make the jewelry of one or two materials each; the last one
// distributes every watch.
var supplierDefs = []supplierDef{
	{"PR01", "Platería Hermanos Castillo", "Taxco", 4},
	{"PR02", "Orfebres Unidos de Jalisco", "Guadalajara", 5},
	{"PR03", "Oro Fino del Bajío", "León", 7},
	{"PR04", "Distribuidora Relojera del Norte", "Monterrey", 3},
}

// materialSuppliers index supplierDefs, in the order of materials.
var materialSuppliers = []int{0, 1, 1, 2}

const watchSupplier = 3

var firstNames = []string{
	"Ana", "María", "Guadalupe", "Fernanda", "Sofía", "Daniela", "Valeria", "Mariana", "Gabriela", "Alejandra",
	"José", "Luis", "Juan", "Carlos", "Jorge", "Miguel", "Alejandro", "Ricardo", "Fernando", "Diego",
}

var lastNames = []string{
	"Hernández", "García", "Martínez", "López", "González", "Pérez", "Rodríguez", "Sánchez", "Ramírez", "Cruz",
	"Flores", "Gómez", "Morales", "Vázquez", "Reyes", "Jiménez", "Torres", "Díaz", "Gutiérrez", "Ruiz",
}
