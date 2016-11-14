package apifederation

import (
	"sort"
)

func SortByGroup(servers []*APIServer) [][]*APIServer {
	serversByPriority := ByPriority(servers)
	sort.Sort(serversByPriority)

	ret := [][]*APIServer{}
	for _, curr := range serversByPriority {
		// check to see if we already have an entry for this group
		existingIndex := -1
		for j, groupInReturn := range ret {
			if groupInReturn[0].Spec.Group == curr.Spec.Group {
				existingIndex = j
				break
			}
		}

		if existingIndex >= 0 {
			ret[existingIndex] = append(ret[existingIndex], curr)
			continue
		}

		ret = append(ret, []*APIServer{curr})
	}

	return ret
}

type ByPriority []*APIServer

func (s ByPriority) Len() int      { return len(s) }
func (s ByPriority) Swap(i, j int) { s[i], s[j] = s[j], s[i] }
func (s ByPriority) Less(i, j int) bool {
	return s[i].Spec.Priority < s[j].Spec.Priority
}
