package awstags // import "example.com/awstags"

type tag struct {
	Key   string
	Value string
}

type tags []tag

func tagsFromMap(raw map[string]interface{}) tags {
	var t tags
	for k, rv := range raw {
		t = append(t, tag{
			Key:   k,
			Value: rv.(string),
		})
	}
	return t
}

// Diff for tags is slightly different then set difference, in that it returns
// all current tags as to create and all removed tags or tags where the values
// differ as remove.
func (prior tags) Diff(current tags) (create tags, remove tags) {
	var difference = func(a, b tags) tags {
		mb := make(map[string]tag, len(b))
		for _, x := range b {
			mb[x.Key] = x
		}
		var diff tags
		for _, x := range a {
			if mbv, found := mb[x.Key]; !found || mbv.Value != x.Value {
				diff = append(diff, mbv)
			}
		}
		return diff
	}
	remove = difference(prior, current)
	create = current

	return create, remove
}
