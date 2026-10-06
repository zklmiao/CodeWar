# 高频复习 59 题（按题解类型分类）

## 链表（12 题）

##### [206\. 反转链表](https://leetcode.cn/problems/reverse-linked-list/)
```Go
/**
 * Definition for singly-linked list.
 * type ListNode struct {
 *     Val int
 *     Next *ListNode
 * }
 */
func reverseList(head *ListNode) *ListNode {
    vhead := &ListNode{Val:0,Next:head}
    pre, cur := vhead, head
    for cur != nil && cur.Next != nil {
        cnext := cur.Next
        cur.Next = cnext.Next
        cnext.Next = pre.Next
        pre.Next = cnext
    }
    return vhead.Next
}
```

##### [92\. 反转链表 II](https://leetcode.cn/problems/reverse-linked-list-ii/)
```Go
// 前置题 206反转链表I
func reverseBetween(head *ListNode, left int, right int) *ListNode {
    vhead := &ListNode{0,head}
    pre := vhead
    for i := 1; i < left; i ++ { pre = pre.Next }
    cur := pre.Next
    for i := 0; i < right - left; i ++ {
        cnext := cur.Next
        cur.Next = cnext.Next
        cnext.Next = pre.Next
        pre.Next = cnext
    }
    return vhead.Next
}
```

##### [83\. 删除排序链表中的重复元素](https://leetcode.cn/problems/remove-duplicates-from-sorted-list/description/)
```Go
func deleteDuplicates(head *ListNode) *ListNode {
    cur := head
    for ; cur != nil && cur.Next != nil; {
        if cur.Val == cur.Next.Val {
            cur.Next = cur.Next.Next
        } else { cur = cur.Next }
    }
    return head   
}
```

##### [82\. 删除排序链表中的重复元素 II](https://leetcode.cn/problems/remove-duplicates-from-sorted-list-ii/)
```Go
// 前置 83.删除排序链表中的重复元素
func deleteDuplicates(head *ListNode) *ListNode {
    vhead := &ListNode{0,head}
    for cur := vhead; cur.Next != nil && cur.Next.Next != nil; {
        repeatVal := cur.Next.Val // 记住假设是重复节点的值
        if repeatVal == cur.Next.Next.Val { // 站在重复节点的前一个节点逐个跨过重复节点删除
            for ; cur.Next != nil && repeatVal == cur.Next.Val; { cur.Next = cur.Next.Next }
        } else {cur = cur.Next} // 删除完移动节点
    }
    return vhead.Next
}
```

##### [21\. 合并两个有序链表](https://leetcode.cn/problems/merge-two-sorted-lists/)
```Go
func mergeTwoLists(list1 *ListNode, list2 *ListNode) *ListNode {
    vhead := new(ListNode)
    var prev *ListNode
    prev=vhead
    for list1 != nil && list2 != nil {
        if list1.Val < list2.Val {
            prev.Next = list1
            list1=list1.Next
        } else {
            prev.Next = list2
            list2=list2.Next
        }
        prev = prev.Next
    }
    if list1 == nil {
        prev.Next = list2
    } else {
        prev.Next = list1
    }
    return vhead.Next
}
```

##### [2\. 两数相加](https://leetcode.cn/problems/add-two-numbers/)
```Go
func addTwoNumbers(l1 *ListNode, l2 *ListNode) *ListNode {
    sum, carry := 0,0
    vhead := &ListNode{}
    var prev *ListNode = vhead
    for l1 != nil || l2 != nil {
        x,y := 0,0
        if (l1 != nil) {
            x = l1.Val
            l1 = l1.Next
        }
        if (l2 != nil) {
            y = l2.Val
            l2 = l2.Next
        }
        sum = x + y + carry
        carry = (sum / 10)
        sum = (sum % 10)
        prev.Next = &ListNode{Val:sum}
        prev = prev.Next
    }
    if (carry > 0) { prev.Next = &ListNode{Val:carry}  }
    return vhead.Next;
}
```

##### [141\. 环形链表](https://leetcode.cn/problems/linked-list-cycle/)
```Go
func hasCycle(head *ListNode) bool {
    if (head == nil || head.Next == nil) {
        return false
    }
    slowPtr, fastPtr := head, head.Next
    for slowPtr != fastPtr {
        if fastPtr == nil || fastPtr.Next == nil {
            return false
        }
        slowPtr = slowPtr.Next
        fastPtr = fastPtr.Next.Next
    }
    return true
}
```

##### [142\. 环形链表 II](https://leetcode.cn/problems/linked-list-cycle-ii/)
```Go
// 前置 141. 环形链表
func detectCycle(head *ListNode) *ListNode {
    if (head == nil || head.Next == nil) { return false }
    slow, fast := head,head
    for fast != nil {
        slow = slow.Next
        if fast.Next == nil {return nil}
        fast = fast.Next.Next
        if (fast == slow) {
            fast = head
            for fast != slow {
                fast = fast.Next
                slow = slow.Next
            }
            return slow
        }
    }
    return nil
}
```

##### [19\. 删除链表的倒数第 N 个结点](https://leetcode.cn/problems/remove-nth-node-from-end-of-list/)
```Go
func removeNthFromEnd(head *ListNode, n int) *ListNode {
    vhead := &ListNode{Val:0,Next:head}
    cur, pre := vhead, head
    for ; n > 0 && pre != nil; n -- {
        pre = pre.Next
    }
    for pre != nil {
        pre = pre.Next
        cur = cur.Next
    }
    cur.Next = cur.Next.Next
    return vhead.Next   
}
```

##### [160\. 相交链表](https://leetcode.cn/problems/intersection-of-two-linked-lists/)
```Go
func getIntersectionNode(headA, headB *ListNode) *ListNode {
    curA, curB := headA,headB
    lenA,lenB := 1,1
    for curA != nil || curB != nil {
        if curA != nil { curA = curA.Next; lenA ++ }
        if curB != nil { curB = curB.Next; lenB ++ }
    }
    var f func (linkA,linkB *ListNode,delta int) *ListNode
    f = func (linkA,linkB *ListNode,delta int) *ListNode {
        for ;delta > 0; delta-- {linkA = linkA.Next}
        for linkA != nil || linkB != nil {
            if linkA == linkB { return linkA } else {
                linkA = linkA.Next
                linkB = linkB.Next
            }
        }
        return nil
    }
    if lenA > lenB {return f(headA,headB,lenA - lenB)} else {return f(headB,headA,lenB - lenA)}
}
```

##### [148\. 排序链表](https://leetcode.cn/problems/sort-list/)
```Go
func sortList(head *ListNode) *ListNode {
    if head == nil { return nil }
    arr := []int{}
    for cur := head; cur != nil; cur = cur.Next { arr = append(arr,cur.Val) }
    slices.Sort(arr)
    for i,cur := 0, head; i < len(arr) && cur != nil; i, cur = i + 1, cur.Next{
        cur.Val = arr[i]
    }
    return head
}
```

##### [143\. 重排链表](https://leetcode.cn/problems/reorder-list/)
```Go
func reorderList(head *ListNode)  {
    if head == nil || head.Next == nil { return }
    cur := head
    list, newlist := []*ListNode{},[]*ListNode{}
    for cur != nil {
        list = append(list,cur)
        cur = cur.Next
    }
    l, r := 0, len(list)  - 1
    for l <= r {
        newlist = append(newlist,list[l])
        l ++
        newlist = append(newlist,list[r])
        r --
    }
    for i := 0; i < len(newlist) - 1; i ++ {
        newlist[i].Next = newlist[i + 1]
        newlist[i + 1].Next = nil
    }
}
```

##### [146\. LRU 缓存](https://leetcode.cn/problems/lru-cache/)
```Go
type entry struct {
    key, val int
}

type LRUCache struct {
    cap int
    mp map[int]*list.Element // key -> *entry
    cacheList *list.List
}

func Constructor(capacity int) LRUCache {
    return LRUCache { capacity, map[int]*list.Element{}, list.New() }
}

func (this *LRUCache) Get(key int) int {
    if v,ok := this.mp[key]; ok {
        this.cacheList.MoveToFront(v); return v.Value.(*entry).val 
    } else { return - 1}
}

func (this *LRUCache) Put(key int, value int)  {
    if e,ok := this.mp[key]; ok {
        e.Value.(*entry).val = value
        this.cacheList.MoveToFront(e);
    } else {
        if len(this.mp) >= this.cap {
            if tail := this.cacheList.Back(); tail != nil {
                delete(this.mp,tail.Value.(*entry).key)
                this.cacheList.Remove(tail)
            }
        }
        this.mp[key] = this.cacheList.PushFront(&entry{key,value})
    }
}

/**
 * Your LRUCache object will be instantiated and called as such:
 * obj := Constructor(capacity);
 * param_1 := obj.Get(key);
 * obj.Put(key,value);
 */
```

## 数组与双指针 哈希表（9 题）

##### [88\. 合并两个有序数组](https://leetcode.cn/problems/merge-sorted-array/)
```go
func merge(nums1 []int, m int, nums2 []int, n int)  {
    p1,p2,sorted := 0,0, []int{}
    for {
        if p1 == m { sorted = append(sorted,nums2[p2:]...); break }
        if p2 == n {sorted = append(sorted,nums1[p1:]...); break }
        if nums1[p1] < nums2[p2] {
            sorted = append(sorted,nums1[p1]); p1 ++ 
        } else {
            sorted = append(sorted,nums2[p2]); p2 ++ 
        }
    }
    copy(nums1,sorted) // 将答案拷贝会ans 数组共oj 读取判断
}
```

##### [1\. 两数之和](https://leetcode.cn/problems/two-sum/)
```Go
func twoSum(nums []int, target int) []int {
    mp := map[int]int{} // v -> rank
    for i,v := range nums {
        if _, ok := mp[target - v]; ok { return []int{i,mp[target-v]} }
        mp[v] = i
    }   
    return []int{}
}
```

##### [15\. 三数之和](https://leetcode.cn/problems/3sum/)
```Go
// 前置 1. 两数之和
// 规则解释，不允许重复使用元素，但允许元素的值可以是相同的
// 比如 -1,-1,2 -> -1+-1+2 = 0; 
// mp[num[j]] = j; i:0, 比较前一个去重, j : i + 1, 比较后一个去重cmp(j,j+1)
func threeSum(nums []int) (ans [][]int) {
    slices.Sort(nums)
    for i := 0; i < len(nums) - 2; i ++ {
        if i > 0 && nums[i - 1] == nums[i] { continue } // 去重
        mp,target := map[int]int{}, -nums[i] // val:num[j] -> rank j
        for j := i + 1; j < len(nums); j ++{
            complete := target - nums[j]
            if _,ok := mp[complete]; ok { // 存在互补的数 才开始收获解，去重
                ans = append(ans,[]int{nums[i],nums[j],complete})
                for ; j+1 < len(nums) && nums[j] == nums[j+1]; j ++ {} // 去重
            } 
            mp[nums[j]] = j
        }
    }
    return
}
```

##### [31\. 下一个排列](https://leetcode.cn/problems/next-permutation/)
```Go
func nextPermutation(nums []int)  {
    if len(nums) <= 1 { return }
    i,j,k := len(nums) - 2, len(nums) - 1, len(nums) - 1
    for ; i >= 0 && nums[i] >= nums[j];i,j = i - 1,j -1 {} // 找第个v 的右边 / 
    // i >= 0  表示不是最后一个排列
    // eg: 12345678 -> 87654321[最后一个字典顺序排列, i 一定会 = -1]
    if i >= 0 {
        for ;nums[k] <= nums[i] && k >= 0; k -- {} // 找到第一个比nums[i]大的点
        nums[i],nums[k] = nums[k],nums[i]
    } // 或者 i,j = j,len(nums) -1; // 因为 j 在的右边i的下一个就是j
    for i,j = i+1,len(nums) -1; i < j; i,j = i + 1,j - 1{ nums[i],nums[j] = nums[j],nums[i] }
}
```

##### [912\. 排序数组](https://leetcode.cn/problems/sort-an-array/)
```Go
// 1. 核心思想与变量定义
// 代码在分区（Partition）时，维护了三个指针，将数组划分为四个区域：
// pivot：基准值，选取左端点 a[l]。
// lt（less than）：指向“小于区”的末尾。[l, lt-1] 区间内的元素都 小于 pivot。
// i：当前遍历指针。[lt, i-1] 区间内的元素都 等于 pivot。
// gt（greater than）：指向“大于区”的开头。[gt+1, r] 区间内的元素都 大于 pivot。
// [i, gt]：是待扫描的未处理区域。

// 2. 循环体执行逻辑（while (i <= gt)）
// 循环内部分三种情况处理：
// a[i] < pivot：把当前元素交换到“小于区”末尾（lt），lt 和 i 同时右移。
// a[i] > pivot：把当前元素交换到“大于区”开头（gt），gt 左移。
// a[i] == pivot：直接跳过，i 右移。
// 当循环结束时，数组被完整分成了 左(< pivot) | 中(== pivot) | 右(> pivot) 三块。

// 3. 递归细节
// 注意递归时传入的范围：
// 左半部分：(l, lt - 1) —— 只处理小于基准的部分。
// 右半部分：(gt + 1, r) —— 只处理大于基准的部分。

// 巧妙之处：中间所有等于 pivot 的元素被集体跳过了。
// 如果数组全是相同元素（如 [2,2,2,2]），lt 永远不动，gt 永远不动，i 一路扫到底，递归时左右区间为空
// 这个退出条件 i <= gt 是整个三路快排最精妙、也最容易写错的地方。它绝对不能用 i < r 或 i < gt 代替。
// 要理解这一点，你需要时刻记住循环过程中的四个区间划分（不变量）：
// 区间范围 状态    含义
// [l, lt-1]    已处理  全部 小于 pivot
// [lt, i-1]    已处理  全部 等于 pivot
// [i, gt]  未处理  待扫描的未知元素
// [gt+1, r]    已处理  全部 大于 pivot
// 循环的任务，就是不断扫描 未处理区间 [i, gt]，直到把它“榨干”为止。
// 1. 为什么必须是 i <= gt？（数学逻辑）
// 当循环执行到某一刻：
// 如果 i == gt，意味着 未处理区间只剩最后gt边界了
// i <= gt 保证了最后一个未知元素也被处理，同时利用 gt 作为动态右墙，精准地避开了已经归类的“大于区”。
// 这是荷兰国旗算法正确运行的生命线。
func quickSort(nums []int,l,r int) {
    if l >= r { return }
    lt,gt,i,privot := l,r,l,nums[l]
    for i <= gt {
        if nums[i] < privot {
            nums[i], nums[lt] = nums[lt],nums[i]
            lt ++; i++
        } else if nums[i] > privot {
            nums[i],nums[gt] = nums[gt],nums[i]
            gt --
        } else { i++ }
    }
    quickSort(nums, l, lt -1)
    quickSort(nums, gt + 1, r)
}

func sortArray(nums []int) []int {
    quickSort(nums,0,len(nums) - 1)
    return nums
}
```

##### [162\. 寻找峰值](https://leetcode.cn/problems/find-peak-element/)
```Go
// 数组中的最大值一定满足题目要求的峰值 
// 包括用例类型 [1,2,6] [6,1,2] func findPeakElement(nums []int) (ans int) , ans 默认初始化0
func findPeakElement(nums []int) (ans int) {
    for i,v := range nums{ if nums[ans] < v {ans = i} }
    return 
}
```

##### [169\. 多数元素](https://leetcode.cn/problems/majority-element/)
```Go
// 给定长度n数组,返回其中的多数元素。多数元素是指在数组中出现次数 大于 ⌊ n/2 ⌋ 的元素
func majorityElement(nums []int) int { // 投票法
    count,candidcate := 0,0
    for _,v := range nums {
        if count == 0 { candidcate = v }
        if v == candidcate { count ++ } else { count -- }
    }
    return candidcate
}
```

##### [215\. 数组中的第K个最大元素](https://leetcode.cn/problems/kth-largest-element-in-an-array/)
```Go
// 利用小根堆container/heap 接口实现小根堆
// 小根堆中只维持k 个元素,超过时pop, 遍历结束堆顶就是第k大元素
// 接口 Less(i,j int) Swap(i,j int) 传入参数都是 rank, minheap-> Less <
type MinHeap []int
func (h *MinHeap) Len() int { return len(*h) }
func (h *MinHeap) Less(i,j int) bool { return (*h)[i] < (*h)[j] }
func (h *MinHeap) Swap(i,j int) { (*h)[i], (*h)[j] = (*h)[j], (*h)[i]}
func (h *MinHeap) Push (v any) { *h = append(*h,v.(int)) }
func (h *MinHeap) Pop () any { v := (*h)[len(*h) - 1]; *h = (*h)[:len(*h) - 1]; return v }
func findKthLargest(nums []int, k int) int {
    h := &MinHeap{}
    for _,v := range nums {
        heap.Push(h,v)
        if h.Len() > k { heap.Pop(h)}
    }
    return (*h)[0]
}
```

##### [232\. 用栈实现队列](https://leetcode.cn/problems/implement-queue-using-stacks/)
```Go
type MyQueue struct {
    rst []int // read only
    wst []int // write only
}
func Constructor() MyQueue {
    return MyQueue{ []int{},[]int{} }
}
func (this *MyQueue) Push(x int)  {
    this.wst = append(this.wst,x)
}
func (this *MyQueue) Pop() int {
    if len(this.rst) == 0 { // 这里必须等rst 为空再加因为 出队是在rst 切片的最后一个
    // rst  不为空直接appen 会把原来的队列顺序打乱
    // 相当于后面append的一批直接插队到最前面了
        for len(this.wst) > 0 {
            this.rst = append(this.rst,this.wst[len(this.wst) - 1])
            this.wst = this.wst[:len(this.wst) - 1]
        }
    }
    v := this.rst[len(this.rst) - 1]; this.rst = this.rst[:len(this.rst) - 1]
    return v
}
func (this *MyQueue) Peek() int {
    if len(this.rst) == 0 {
        for len(this.wst) > 0 {
            this.rst = append(this.rst,this.wst[len(this.wst) - 1])
            this.wst = this.wst[:len(this.wst) - 1]
        }
    }
    return this.rst[len(this.rst) - 1];
}
func (this *MyQueue) Empty() bool {
    return len(this.rst) == 0 && len(this.wst) == 0
}

/**
 * Your MyQueue object will be instantiated and called as such:
 * obj := Constructor();
 * obj.Push(x);
 * param_2 := obj.Pop();
 * param_3 := obj.Peek();
 * param_4 := obj.Empty();
 */
```

## 字符串处理（6 题）

##### [151\. 反转字符串中的单词](https://leetcode.cn/problems/reverse-words-in-a-string/)
```Go
// strings.Split("a good   example", " ") 坑点 -> ["a", "good", "", "", "example"]
func reverseWords(s string) (ans string) {
    strList := strings.Split(s," ")
    for i := len(strList) - 1; i >= 0; i -- {
        if strList[i] != "" { ans += (strList[i]+" ") }
    }
    return strings.TrimSpace(ans)
}
```

##### [5\. 最长回文子串](https://leetcode.cn/problems/longest-palindromic-substring/)
```Go
func longestPalindrome(s string) string {
    start,maxLen := 0,0
    var expand func (s string, l,r int)
    expand = func (s string, l,r int) { // 中心扩展法 <-l(l-1), (r+1)r->
        for ;l >= 0 && r < len(s) && s[l] == s[r];l,r = l - 1, r + 1 {
            if len(s[l:r + 1]) > maxLen {start = l; maxLen = r - l + 1}
        }
    }
    for i := 0; i < len(s); i ++ { expand(s,i,i); expand(s,i,i + 1) }
    return s[start:start + maxLen]   
}
```

##### [415\. 字符串相加](https://leetcode.cn/problems/add-strings/)
```Go
func addStrings(num1 string, num2 string) string {
    ans := []byte{}
    l,r,carry := len(num1) - 1,len(num2) - 1, 0
    for l >= 0 || r >= 0 || carry > 0 {
        sum := carry
        if l >= 0 {sum += int(num1[l] - '0'); l --}
        if r >= 0 {sum += int(num2[r] - '0'); r --}
        carry = sum/10
        ans = append(ans,byte(sum%10) + '0')
    }
    for l,r = 0,len(ans) - 1; l <= r; l,r = l + 1, r - 1 { ans[l],ans[r] = ans[r],ans[l] }
    return string(ans)
}
```

##### [43\. 字符串相乘](https://leetcode.cn/problems/multiply-strings/)
```Go
// 做两位数乘法
func multiply(num1 string, num2 string) (ans string) {
    if num1 == "0" || num2 == "0" { return "0" }
    arr := make([]int,len(num1) + len(num2))
    for l := len(num1) - 1; l >= 0; l-- {
        a := int(num1[l] - '0')
        for r := len(num2) - 1; r >= 0; r-- {
            b := int(num2[r] - '0')
            sum := a*b + arr[r + l + 1] // 乘积累加上一次的进位
            arr[r+l] += sum / 10 // 两位数乘法，十位进位要累计回上一次的进位
            arr[r + l + 1] = sum % 10 // 各位不需要，直接覆盖
        }
    }
    start := 0
    for ; start < len(arr); start ++ { if arr[start] > 0 {break} }
    for ; start < len(arr); start ++ { ans += string(arr[start]+'0') }
    return
}
```

##### [8\. 字符串转换整数 \(atoi\)](https://leetcode.cn/problems/string-to-integer-atoi/)
```Go
func myAtoi(s string) int {
    s = strings.TrimSpace(s)
    if len(s) == 0 { return 0 }
    i,sign,sum := 0, int64(1), int64(0)
    if s[i] == '-' { sign *= -1; i++ } else if s[i] == '+' { sign = 1; i++ }
    for ; i < len(s) && unicode.IsDigit(rune(s[i])); i ++ {
        sum = sum * 10 + int64(s[i] - '0')
        if sum*sign >= math.MaxInt32 { return math.MaxInt32 }
        if sum*sign <= math.MinInt32 { return math.MinInt32 }
    }
    return int(sum*sign)
}
```

##### [165\. 比较版本号](https://leetcode.cn/problems/compare-version-numbers/)
```Go
func compareVersion(version1 string, version2 string) int {
    for l,r := 0,0;l < len(version1) || r < len(version2);{
        s1,s2 := "",""
        for ;l < len(version1) && version1[l] != '.';l ++  {s1 += string(version1[l]) }
        l ++
        for ;r < len(version2) && version2[r] != '.';r ++  {s2 += string(version2[r]) }
        r ++
        t1,_ := strconv.Atoi(s1); t2,_ := strconv.Atoi(s2)
        if t1 > t2 {return 1} else if t1 < t2 { return -1 }        
    }
    return 0
}
```

## 回溯（4 题）

##### [78\. 子集](https://leetcode.cn/problems/subsets/)
```Go
func subsets(nums []int) (ans [][]int) {
    path := []int{}
    var backtrace func(nums []int,start int)
    backtrace = func(nums []int,start int) {
        // 将当前路径的副本加入结果（因为后续会修改 path）
        ans = append(ans,slices.Clone(path))
        // 从 start 开始，依次尝试添加每个元素
        for ;start < len(nums); start ++ {
            path = append(path,nums[start]) // 选择
            backtrace(nums,start + 1)   // 递归到下一层（只往后选）
            path = path[:len(path) - 1] // 撤销选择（回溯）
        }
    }
    backtrace(nums,0)
    return
}
```

##### [46\. 全排列](https://leetcode.cn/problems/permutations/)
```Go
func permute(nums []int) (ans [][]int) {
    path,used := []int{}, make([]bool, len(nums))
    var backtrace func (nums []int,used []bool)
    backtrace = func (nums []int,used []bool) {
        if len(path) == len(nums) { ans = append(ans,slices.Clone(path)); return }
        for i,v := range nums {
            if used[i] == true { continue }
            used[i] = true; path = append(path, v)
            backtrace(nums,used)
            used[i] = false; path = path[:len(path) - 1]
        }
    }
    backtrace(nums,used)
    return
}
```

##### [22\. 括号生成](https://leetcode.cn/problems/generate-parentheses/)
```Go
func generateParenthesis(n int) (ans []string) {
    path := ""
    var backtrace func (open, close, n int)
    backtrace = func (open, close, n int) {
        if len(path) == n*2 { ans = append(ans,path); return }
        // 根据题目有约束条件: open <= close <= n 
        if open < n {
            path += "("
            backtrace(open+1,close,n)
            path = path[:len(path) - 1]
        }
        if (open > close) { // 这里必须是 if ... 不能写 else if ...
            path += ")"
            backtrace(open,close + 1,n)
            path = path[:len(path) - 1]
        }
    }
    backtrace(0,0,n)
    return 
}

```

##### [93\. 复原 IP 地址](https://leetcode.cn/problems/restore-ip-addresses/)
```Go
func restoreIpAddresses(s string) (ans []string) {
    path := ""
    var backtrace func (s string, start, seg int)
    backtrace = func (s string, start, seg int) {
        if seg == 4 {if start == len(s) { ans = append(ans,path[:len(path) - 1]) }; return}
        for partLen := 1 ; partLen <= 3 && start + partLen <= len(s); partLen ++ {
            strV := s[start:start+partLen]; v,_ := strconv.Atoi(strV)
            if (partLen > 1 && strV[0] == '0') || v > 255 { break }
            path += (strV + ".")
            backtrace(s, start + partLen,seg + 1)
            path = path[:len(path) - partLen - 1]
        }
    }
    backtrace(s,0,0)
    return
}
```

## 动态规划-一维（7 题）

##### [70\. 爬楼梯](https://leetcode.cn/problems/climbing-stairs/)
```Go
func climbStairs(n int) int {
    dp := [3]int{1,2,0} // 初始化[0] [1]: 每次往上可以爬一格或者两格
    for i := 3 ; i <= n; i ++ {
        dp[2] = dp[0] + dp[1]
        dp[0] = dp[1]
        dp[1] = dp[2]
    }
    if n < 3 { return dp[n - 1] } else { return dp[2] }
}
```

##### [53\. 最大子数组和](https://leetcode.cn/problems/maximum-subarray/)
```Go
func maxSubArray(nums []int) int {
    sum,ans := 0,nums[0]
    for _,v := range nums {
        sum = max(sum + v,v)
        ans = max(sum,ans)
    }
    return ans
}
```

##### [121\. 买卖股票的最佳时机](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock/)
```Go
// profit：最大利润。遍历过程中，不断更新为到目前为止能获得的最大利润
// purcahse：历史最低股价。记录遍历到当前天为止，出现过的最低股票价格
// 只能买卖一次股票，在给定的股票价格数组里: 
// 有机会就交易并更新交易利润记录，最后留下的是某次交易的最大利润
func maxProfit(prices []int) int {
    purchase, ans, i := prices[0],0,0
    
    for ; i < len(prices); i ++ { 
        purchase = min(purchase,prices[i])
        ans = max(ans, prices[i] - purchase)
    }
    return ans
}
```

##### [122\. 买卖股票的最佳时机 II](https://leetcode.cn/problems/best-time-to-buy-and-sell-stock-ii/)
```Go
// 题目意思: 周期内无限次买卖 [低买高卖同价不管]
// 低价买入将当前价格作为新的成本，下次价格高于成本时卖出，累计利润更新这次股价为持有成本
func maxProfit(prices []int) int {
    purchase, ans, i := prices[0], 0, 0
    for ; i < len(prices); i ++ { 
        if prices[i] < purchase {purchase = prices[i]} else if prices[i] > purchase {
            ans += prices[i] - purchase; purchase = prices[i] } else {}
            // else {} 出售价格和成本价一样 无获益 所以什么也不干
            // if prices[i] > purchase 可以去掉, 变成 出售价格和成本价一样也进行一次买卖交易
    }
    return ans
}
```

##### [198\. 打家劫舍](https://leetcode.cn/problems/house-robber/)
```Go
func rob(nums []int) int {
    if len(nums) < 3 { return max(nums[0],nums[1]) }
    dp := make([]int,len(nums)); dp[0] = nums[0] 
// 今晚收益: max(dp[i - 2] + nums[i] 前天晚上偷的+今天晚上偷的收益, dp[i - 1] 只偷昨晚的收益)
    for i := 2; i < len(nums); i++ { dp[i] = max(dp[i - 2] + nums[i],dp[i - 1]) }
    return dp[len(nums) - 1]
}
```

##### [322\. 零钱兑换](https://leetcode.cn/problems/coin-change/)
```Go
func coinChange(coins []int, amount int) int {
    dp := make([]int,amount + 1)
    for i := 1; i < amount + 1; i ++ {dp[i] = amount + 1}
    for i := 0; i < len(coins); i ++ {
        for j := coins[i]; j <= amount; j ++ {
            dp[j] = min(dp[j], dp[j - coins[i]] + 1)
        }
    }
    if dp[amount] == amount + 1 {return - 1} else { return dp[amount] }
}
```

##### [300\. 最长递增子序列](https://leetcode.cn/problems/longest-increasing-subsequence/)
```Go
func lengthOfLIS(nums []int) int {
    ans,dp := 0, make([]int, len(nums))
    for i := 0; i < len(dp); i ++ { dp[i] = 1 }
    for i := 0; i < len(dp); i ++ {
        for j := 0; j < i; j ++ { // j ->,..,i 双指针 i 先走,  j往i 移动
            if nums[j] < nums[i] {dp[i] = max(dp[i],dp[j] + 1)}
            if ans < dp[i] { ans = dp[i] }
        }
    }
    return ans
}
```

## 动态规划-二维（3 题）

##### [64\. 最小路径和](https://leetcode.cn/problems/minimum-path-sum/)
```Go
func minPathSum(grid [][]int) int {
    for i := 1; i < len(grid); i ++ {grid[i][0] += grid[i - 1][0]}
    for i := 1; i < len(grid[0]); i ++ {grid[0][i] += grid[0][i-1]}
    for i := 1; i < len(grid); i ++{
        for j := 1; j < len(grid[0]); j ++ {
            grid[i][j] += min(grid[i - 1][j], grid[i][j - 1])
        }
    }
    return grid[len(grid) - 1][len(grid[0]) - 1]
}
```

##### [1143\. 最长公共子序列](https://leetcode.cn/problems/longest-common-subsequence/)
```Go
func longestCommonSubsequence(text1 string, text2 string) int {
    dp:= make([][]int, len(text1)+1)
    for i := 0; i < len(text1) + 1; i ++ { dp[i] = make([]int,len(text2) + 1) }
    for i := 1; i < len(text1) + 1; i ++ {
        for j :=1; j < len(text2) + 1; j ++ {
            if text1[i - 1] == text2[j - 1] { dp[i][j] = dp[i - 1][j - 1] + 1 } else {
                dp[i][j] = max(dp[i-1][j],dp[i][j-1])
            }
        }
    }
    return dp[len(text1)][len(text2)]
}
```

##### [72\. 编辑距离](https://leetcode.cn/problems/edit-distance/)
```Go
func minDistance(word1 string, word2 string) int {
    if word1 == word2 {return 0}
    dp:= make([][]int, len(word1)+1)
    for i := 0; i < len(word1) + 1; i ++ { dp[i] = make([]int,len(word2) + 1); dp[i][0] = i }
    for i := 0; i < len(word2) + 1; i ++ { dp[0][i] = i }
    for i := 1; i < len(word1) + 1; i ++ {
        for j := 1; j < len(word2) + 1; j ++ {
            if word1[i - 1] == word2[j - 1] { dp[i][j] = dp[i - 1][j - 1] } else {
                dp[i][j] = min(dp[i - 1][j], dp[i - 1][j - 1], dp[i][j - 1]) + 1
            }
        }
    }
    return dp[len(word1)][len(word2)]
}
```

## 二分查找（3 题）

##### [704\. 二分查找](https://leetcode.cn/problems/binary-search/) / [35\. 搜索插入位置](https://leetcode.cn/problems/search-insert-position/)
```Go
// https://leetcode.cn/problems/search-insert-position/ 另一题
// 704
func search(nums []int, target int) int {
    l,r,mid := 0, len(nums) - 1, 0
    for l <= r {
        mid = (l + r) >> 1
        if (target == nums[mid]) { 
            return mid 
        } else if (target < nums[mid]) {
           r = mid - 1 
        } else {
           l = mid + 1
        }
    }
    return -1
}
// 35
func searchInsert(nums []int, target int) int {
    l,r,mid := 0, len(nums) - 1, 0
    for l <= r {
        mid = (l + r) >> 1
        if (target <= nums[mid]) {
           r = mid - 1 
        } else {
           l = mid + 1
        }
    }
    return l
}
```

##### [34\. 在排序数组中查找元素的第一个和最后一个位置](https://leetcode.cn/problems/find-first-and-last-position-of-element-in-sorted-array/)
```Go
func searchRange(nums []int, target int) []int {
    l,r,mid := 0,len(nums) - 1, 0
    for l <= r {
        mid = (r + l) >> 1
        if target <= nums[mid] { r = mid - 1}
        if target > nums[mid] { l = mid + 1}
    }
    // 上面的l 有三种可能 0 <= l <= len(nums)
    // 0: target 小于数组中的任意值或者第一个就是， 
    // len(nums): 大于所有值， 0 <= l < len(nums) 在nums 数组区间中，但不存在该元素
    ans := []int{l,l}
    if l >= len(nums) || nums[l] != target { return []int{-1,-1} }
    for i := l; i < len(nums); i ++ {
        if target == nums[i] {ans[1] = i} else {break}
    }   
    return ans
}
```

##### [33\. 搜索旋转排序数组](https://leetcode.cn/problems/search-in-rotated-sorted-array/)
```Go
func search(nums []int, target int) int {
    l,r,mid := 0, len(nums) - 1, 0
    for l <= r {
        mid = (l + r) >> 1
        if nums[mid] == target { return mid } // 不管是否旋转都符合二分查找规律:nums [l,mid),mid,(mid,r]
        if target < nums[mid] { // 左半边区间: target < nums[mid] && target >= nums[l]
            if target < nums[mid] && target >= nums[l] { r = mid - 1 } else { l = mid + 1 }
        } else {               //  右半边区间: target > nums[mid] && target <= nums[r]
            if target > nums[mid] && target <= nums[r] { l = mid + 1 } else { r = mid - 1}
        }
    }
    return -1
}
```

## 位运算与数学（1 题）

##### [69\. x 的平方根 ](https://leetcode.cn/problems/sqrtx/)
```Go
// 牛顿迭代公式 求开根:
//r初始：r = x / 2; r*r <= x 停止迭代; 迭代公式 r = (r + x/r)/2; 迭代结束返回r 即为x的开平方根
func mySqrt(x int) int {
    if x < 2 { return x }; r := x/2
    for r * r > x { r = (r + x / r)/2 }; return r
}
```

## 栈与队列（1 题）

##### [20\. 有效的括号](https://leetcode.cn/problems/valid-parentheses/)
```Go
func isValid(s string) bool {
    mp := map[rune]rune{'(':')','[':']','{':'}'}
    st := []rune{}
    for _,c := range s {
        if len(st) == 0 {
            st = append(st, c)
        } else if mp[st[len(st) - 1]] == c {
            st = st[:len(st) - 1]
        } else { st = append(st, c) }
    }
    return len(st) == 0
}
```

## 滑动窗口（2 题）

##### [209\. 长度最小的子数组](https://leetcode.cn/problems/minimum-size-subarray-sum/)
```Go
func minSubArrayLen(target int, nums []int) int {
    sum,minLen := 0, 1000000
    for l,r := 0,0; r < len(nums); r ++ {
        sum += nums[r]
        for target <= sum {
            sum -= nums[l]; minLen = min(minLen, r - l + 1); l++
        }
    }
    if minLen == 1000000 { return 0 } else { return minLen }
}
```

##### [3\. 无重复字符的最长子串](https://leetcode.cn/problems/longest-substring-without-repeating-characters/)
```Go
func lengthOfLongestSubstring(s string) int {
    ans,mp := 0, map[byte]bool{}
    for l,r := 0,0; r < len(s); r ++ {
        for mp[s[r]] {delete(mp,s[l]); l ++ }
        ans = max(ans,r - l + 1); mp[s[r]] = true
    }
    return ans
}
```

## 贪心（1 题）

##### [56\. 合并区间](https://leetcode.cn/problems/merge-intervals/)
```Go
// sort.Slice(intervals, func(i, j int) bool {
//      return intervals[i][0] < intervals[j][0]
//  })
// 按第一个值从小到大排序
// 从前往后遍历加入到res, 发现当前加入的vector<int> 左边界a[i][0] <=res.back()[1]末尾 合并区间，
// 将res.back()[1]更新为max(新加入a[0][1], back()[1]) 大者
func merge(intervals [][]int) (ans [][]int) {
    sort.Slice(intervals,func (i,j int) bool {
        return intervals[i][0] < intervals[j][0]
    })
    ans = append(ans,intervals[0])
    for i := 1; i < len(intervals); i ++ {
        if ans[len(ans) - 1][1] >= intervals[i][0] {
            ans[len(ans) - 1][1] = max(ans[len(ans) - 1][1], intervals[i][1])
        } else { ans = append(ans,intervals[i]) }
    }
    return ans
}
```

## 矩阵与模拟（1 题）

##### [54\. 螺旋矩阵](https://leetcode.cn/problems/spiral-matrix/)
```Go
func spiralOrder(matrix [][]int) (ans []int) {
    top,r,bottom,l := 0,len(matrix[0]) - 1, len(matrix) - 1, 0 
    for {
        for i := l; i <= r; i ++ {ans = append(ans, matrix[top][i]) }
        if top++; top > bottom { break }
        for i := top; i <= bottom; i ++ {ans = append(ans, matrix[i][r]) }
        if r--; r < l { break }
        for i := r; i >= l ; i -- {ans = append(ans, matrix[bottom][i]) }
        if bottom--; bottom < top { break }
        for i := bottom; i >= top ; i -- {ans = append(ans, matrix[i][l]) }
        if l++; l > r { break }
    }
    return
}
```

## 图（1 题）

##### [200\. 岛屿数量](https://leetcode.cn/problems/number-of-islands/)
```Go
// numIslands 遍历整个网格，遇到 '1' 时岛屿计数 ans 加一，然后调用 dfs 将整座岛屿“沉没”（全部变为 '0'）
// dfs 递归地将当前陆地的上下左右相邻陆地全部标记为 '0'，遇到边界或 '0' 则返回
func numIslands(grid [][]byte) (ans int) {
    var dfs func (grid [][]byte, i,j int)
    dfs = func (grid [][]byte, i,j int) {
        if i < 0 || i >= len(grid) || j < 0 || j >= len(grid[0]) || grid[i][j] == '0' { return }
        grid[i][j] = '0'
        dfs(grid,i - 1,j);dfs(grid,i + 1,j)
        dfs(grid,i,j - 1);dfs(grid,i,j + 1)
    }
    for i:= 0; i < len(grid); i++ {
        for j:= 0; j < len(grid[i]); j++ {
            if grid[i][j] == '1' { ans ++;dfs(grid,i,j) }
        }
    }
    return
}
```

## 树-递归与遍历（5 题）

##### [94\. 二叉树的中序遍历](https://leetcode.cn/problems/binary-tree-inorder-traversal/)
```Go
/**
 * Definition for a binary tree node.
 * type TreeNode struct {
 *     Val int
 *     Left *TreeNode
 *     Right *TreeNode
 * }
 */
func inorderTraversal(root *TreeNode) (ans []int) {
    var travel func (root *TreeNode)
    travel = func (root *TreeNode) {
        if root == nil  { return }
        travel(root.Left)
        ans = append(ans,root.Val)
        travel(root.Right)
    }
    travel(root);
    return
}
```

##### [104\. 二叉树的最大深度](https://leetcode.cn/problems/maximum-depth-of-binary-tree/)
```Go
func maxDepth(root *TreeNode) int {
    var travel func (root* TreeNode) int
    travel = func (root* TreeNode) int {
        if root == nil { return 0}
        left := travel(root.Left)
        right := travel(root.Right)
        return max(left,right) + 1
    }
    return travel(root)
}
```

##### [226\. 翻转二叉树](https://leetcode.cn/problems/invert-binary-tree/)
```Go
func invertTree(root *TreeNode) *TreeNode {
    var travel func (root *TreeNode) *TreeNode
    travel = func (root *TreeNode) *TreeNode {
        if root == nil {return nil}
        root.Left,root.Right = root.Right, root.Left
        travel(root.Left); travel(root.Right)
        return root 
     }
    return travel(root)
}
```

##### [236\. 二叉树的最近公共祖先](https://leetcode.cn/problems/lowest-common-ancestor-of-a-binary-tree/)
```Go
func lowestCommonAncestor(root, p, q *TreeNode) *TreeNode {
    var travel func (root, p, q *TreeNode) *TreeNode
    travel = func (root, p, q *TreeNode) *TreeNode {
        if root == nil || root == p || root == q {return root}
        l,r := travel(root.Left,p,q),travel(root.Right,p,q) //注意:要获取root 的l,r子树才能做下面的比较
        if l == nil && r == nil { return nil } // 都为空返回nil,不返回root!
        if l != nil && r == nil { return l }
        if l == nil && r != nil { return r }
        return root
    }
    return travel(root,p,q)
}
```

##### [105\. 从前序与中序遍历序列构造二叉树](https://leetcode.cn/problems/construct-binary-tree-from-preorder-and-inorder-traversal/)
```Go
 // preorder: [ 根节点, [左子树的前序遍历结果], [右子树的前序遍历结果] ]
 // inorder: [ [左子树的中序遍历结果], 根节点, [右子树的中序遍历结果] ]
 // 递归分割序列代入递归构建
func buildTree(preorder []int, inorder []int) *TreeNode {
    var travel func (preorder, inorder []int) *TreeNode
    travel = func (preorder, inorder []int) *TreeNode {
        if len(preorder) == 0 {return nil}
        leftSize := slices.Index(inorder, preorder[0])
        // preorder[1:leftSize + 1]: 
        // 先序左子树的数据数量是 leftSize,切片取值rank[1,leftSize + 1) 左闭右开，所以 leftSize 需要+ 1
        // inorder[:leftSize] 在pre[0]前取左子树数据
        l := travel(preorder[1:leftSize + 1],inorder[:leftSize]) 
        // 右子树，都在构建的root节点切片数据右边，所以leftSize + 1 跳过 这个节点
        r := travel(preorder[leftSize + 1:],inorder[leftSize + 1:]) 
        return &TreeNode{preorder[0],l,r}
    }
    return travel(preorder,inorder)
}
```

## 树-BFS层序（3 题）

##### [102\. 二叉树的层序遍历](https://leetcode.cn/problems/binary-tree-level-order-traversal/)
```Go
func levelOrder(root *TreeNode) (ans [][]int) {
    if root == nil { return }
    cur := []*TreeNode{root}
    var travel func ()
    travel = func () {
        arr := []int{}
        for _,node := range cur { arr = append(arr,node.Val) }
        ans = append(ans,arr)
    }
    for len(cur) > 0 {
        next := []*TreeNode{}
        travel()
        for _,node := range cur {
            if node.Left != nil { next = append(next, node.Left) }
            if node.Right != nil { next = append(next, node.Right) }
        }
        cur = next
    }
    return 
}
```

##### [103\. 二叉树的锯齿形层序遍历](https://leetcode.cn/problems/binary-tree-zigzag-level-order-traversal/)
```Go
func zigzagLevelOrder(root *TreeNode) (ans [][]int) {
    if root == nil { return }
   onLeft := true
    cur := []*TreeNode{ root }
    var travel func ()
    travel = func () {
        arr := []int{}
        if onLeft {
            for _,node := range cur { arr = append(arr,node.Val) }
        } else {
            for i := len(cur) - 1; i >= 0; i-- {  arr = append(arr,cur[i].Val) }
        }
        onLeft = !onLeft
        ans = append(ans, arr) 
    }
    for len(cur) > 0 {
        next := []*TreeNode{}
        travel()
        for _,node := range cur {
            if node.Left != nil { next = append(next, node.Left) }
            if node.Right != nil { next = append(next, node.Right) }
        }
        cur = next
    }
    return    
}
```

##### [199\. 二叉树的右视图](https://leetcode.cn/problems/binary-tree-right-side-view/)
```Go
func rightSideView(root *TreeNode) (ans []int) {
    if root == nil { return }
    var travel func (nodes []*TreeNode)
    travel = func (nodes []*TreeNode) { ans = append(ans,nodes[len(nodes) - 1].Val) }
    cur := []*TreeNode{root}
    for len(cur) > 0 {
        next := []*TreeNode{}
        travel(cur)
        for _,v := range cur {
            if v.Left != nil { next = append(next,v.Left) }
            if v.Right != nil { next = append(next,v.Right) }
        }
        cur = next
    }
    return
}
```
