package oci

type Tag struct {
	Host      string
	Name      string
	Namespace string
	Version   string
}

func (t *Tag) String() string {
	if t.Namespace != "" {
		return t.Host + "/" + t.Namespace + "/" + t.Name + ":" + t.Version
	}

	if t.Host != "" {
		return t.Host + "/" + t.Name + ":" + t.Version
	}

	return t.Name + ":" + t.Version
}

func (t *Tag) NamespacedName() string {
	if t.Namespace != "" {
		return t.Namespace + "/" + t.Name
	}

	return t.Name
}
