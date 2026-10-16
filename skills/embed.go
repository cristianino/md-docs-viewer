// Package skills embebe el skill de mdocs para agentes (formato Agent Skills:
// una carpeta con SKILL.md). Lo instala y empaqueta "mdocs skill".
package skills

import _ "embed"

//go:embed mdocs/SKILL.md
var MdocsSkill []byte

// Name es el nombre del skill (y de su carpeta).
const Name = "mdocs"
