package knowledge

import "testing"

func TestSearch(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name        string
		query       string
		wantTopName string // exercise name expected at the top of results
		wantNoHits  bool   // true if Search should return nil
	}{
		{
			name:        "direct exercise name match",
			query:       "помоги решить pointone",
			wantTopName: "pointone",
		},
		{
			name:        "display name in query",
			query:       "как написать функцию PointOne",
			wantTopName: "pointone",
		},
		{
			name:        "concept keyword surfaces specific exercise",
			query:       "что такое тройной указатель",
			wantTopName: "ultimatepointone",
		},
		{
			name:        "swap by russian description",
			query:       "поменять местами значения двух int",
			wantTopName: "swap",
		},
		{
			name:        "fibonacci concept",
			query:       "напиши фибоначчи",
			wantTopName: "fibonacci",
		},
		{
			name:        "prime concept",
			query:       "проверить простое число",
			wantTopName: "isprime",
		},
		{
			name:       "unrelated query returns nothing",
			query:      "расскажи про погоду в москве",
			wantNoHits: true,
		},
		{
			name:       "empty query is safe",
			query:      "",
			wantNoHits: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := Search(tc.query, 3)
			if tc.wantNoHits {
				if got != nil {
					t.Fatalf("expected no hits, got %d: %+v", len(got), got)
				}
				return
			}
			if len(got) == 0 {
				t.Fatalf("expected at least one hit for %q, got none", tc.query)
			}
			if got[0].Exercise.Name != tc.wantTopName {
				t.Fatalf("for %q want top=%q, got %q (score=%d). all results: %+v",
					tc.query, tc.wantTopName, got[0].Exercise.Name, got[0].Score, got)
			}
		})
	}
}

func TestFindByName(t *testing.T) {
	t.Parallel()
	if ex := FindByName("pointone"); ex == nil || ex.DisplayName != "PointOne" {
		t.Fatalf("FindByName(pointone) = %v", ex)
	}
	if ex := FindByName("POINTONE"); ex == nil {
		t.Fatalf("FindByName should be case-insensitive")
	}
	if ex := FindByName("nonexistent"); ex != nil {
		t.Fatalf("FindByName(nonexistent) want nil, got %v", ex)
	}
}

func TestExercisesIntegrity(t *testing.T) {
	t.Parallel()
	if len(Exercises) < 30 {
		t.Fatalf("expected at least 30 exercises, got %d", len(Exercises))
	}
	seen := make(map[string]bool)
	for _, ex := range Exercises {
		if ex.Name == "" || ex.DisplayName == "" {
			t.Errorf("exercise has empty name fields: %+v", ex)
		}
		if seen[ex.Name] {
			t.Errorf("duplicate exercise name: %q", ex.Name)
		}
		seen[ex.Name] = true
		if ex.Signature == "" {
			t.Errorf("%s: missing signature", ex.Name)
		}
		if ex.Solution == "" {
			t.Errorf("%s: missing solution", ex.Name)
		}
		if ex.Description == "" {
			t.Errorf("%s: missing description", ex.Name)
		}
	}
}
