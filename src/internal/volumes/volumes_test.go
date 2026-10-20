package volumes

import (
	"runtime"
	"testing"
)

func TestLister(t *testing.T) {
	vs := Lister()
	if len(vs) == 0 {
		t.Fatal("aucun volume trouvé")
	}
	for _, v := range vs {
		if v.Total <= 0 || v.Libre < 0 || v.Libre > v.Total {
			t.Errorf("volume incohérent : %+v", v)
		}
	}
	if Utilise(vs[0].Chemin) <= 0 {
		t.Errorf("espace utilisé de %s non trouvé", vs[0].Chemin)
	}
}

func TestUtiliseHorsRacine(t *testing.T) {
	dossier := "/usr"
	if runtime.GOOS == "windows" {
		dossier = `C:\Windows`
	}
	if u := Utilise(dossier); u != 0 {
		t.Errorf("%s n'est pas une racine de volume, utilisé = %d", dossier, u)
	}
}

func TestNormaliser(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("lettres de lecteur : Windows seulement")
	}
	if normaliser("c:") != normaliser(`C:\`) {
		t.Error("« c: » et « C:\\ » doivent désigner la même racine")
	}
}
