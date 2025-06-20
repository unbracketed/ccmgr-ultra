package git

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewGitOperations(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	ops := NewGitOperations(repo, mockGit)

	assert.NotNil(t, ops)
	assert.Equal(t, repo, ops.repo)
	assert.Equal(t, mockGit, ops.gitCmd)
}

func TestNewGitOperations_NilGitCmd(t *testing.T) {
	repo := createTestRepository()

	ops := NewGitOperations(repo, nil)

	assert.NotNil(t, ops)
	assert.IsType(t, &GitCmd{}, ops.gitCmd)
}

func TestCreateBranch_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify new-feature", fmt.Errorf("unknown revision"))
	mockGit.SetCommand("branch new-feature main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateBranch("new-feature", "main")

	assert.NoError(t, err)
}

func TestCreateBranch_EmptyName(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateBranch("", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestCreateBranch_AlreadyExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify existing-branch", "abc123def")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateBranch("existing-branch", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestCreateBranch_DefaultSource(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify new-branch", fmt.Errorf("unknown revision"))
	mockGit.SetCommand("branch new-branch main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateBranch("new-branch", "")

	assert.NoError(t, err)
}

func TestDeleteBranch_Success(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main" // Ensure we're not on the branch to delete
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("branch -d feature-branch", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteBranch("feature-branch", false)

	assert.NoError(t, err)
}

func TestDeleteBranch_EmptyName(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteBranch("", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestDeleteBranch_CurrentBranch(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteBranch("main", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot delete current branch")
}

func TestDeleteBranch_NotExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))

	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteBranch("nonexistent", false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestDeleteBranch_Force(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature-branch", "abc123def")
	mockGit.SetCommand("branch -D feature-branch", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteBranch("feature-branch", true)

	assert.NoError(t, err)
}

func TestBranchOperations_BranchExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify existing-branch", "abc123def")
	mockGit.SetError("rev-parse --verify nonexistent-branch", fmt.Errorf("unknown revision"))

	ops := NewGitOperations(repo, mockGit)

	assert.True(t, ops.BranchExists("existing-branch"))
	assert.False(t, ops.BranchExists("nonexistent-branch"))
}

func TestGetBranchInfo_Success(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("rev-parse main", "abc123def")
	mockGit.SetCommand("rev-parse --abbrev-ref main@{upstream}", "origin/main")
	mockGit.SetCommand("rev-list --left-right --count main...origin/main", "2\t1")

	ops := NewGitOperations(repo, mockGit)

	info, err := ops.GetBranchInfo("main")

	require.NoError(t, err)
	assert.Equal(t, "main", info.Name)
	assert.True(t, info.Current)
	assert.Equal(t, "abc123def", info.Head)
	assert.Equal(t, "origin/main", info.Upstream)
	assert.Equal(t, "origin", info.Remote)
	assert.Equal(t, 2, info.Ahead)
	assert.Equal(t, 1, info.Behind)
}

func TestGetBranchInfo_DefaultToCurrent(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "feature"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("rev-parse feature", "def456ghi")
	mockGit.SetError("rev-parse --abbrev-ref feature@{upstream}", fmt.Errorf("no upstream"))

	ops := NewGitOperations(repo, mockGit)

	info, err := ops.GetBranchInfo("")

	require.NoError(t, err)
	assert.Equal(t, "feature", info.Name)
	assert.True(t, info.Current)
}

func TestGetBranchInfo_NotExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))

	ops := NewGitOperations(repo, mockGit)

	_, err := ops.GetBranchInfo("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestListBranches(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	branchOutput := `* main
  feature-1
  feature-2`

	mockGit.SetCommand("branch", branchOutput)
	
	// Mock GetBranchInfo calls for each branch
	mockGit.SetCommand("rev-parse --verify main", "abc123")
	mockGit.SetCommand("rev-parse main", "abc123")
	mockGit.SetError("rev-parse --abbrev-ref main@{upstream}", fmt.Errorf("no upstream"))
	
	mockGit.SetCommand("rev-parse --verify feature-1", "def456")
	mockGit.SetCommand("rev-parse feature-1", "def456")
	mockGit.SetError("rev-parse --abbrev-ref feature-1@{upstream}", fmt.Errorf("no upstream"))
	
	mockGit.SetCommand("rev-parse --verify feature-2", "ghi789")
	mockGit.SetCommand("rev-parse feature-2", "ghi789")
	mockGit.SetError("rev-parse --abbrev-ref feature-2@{upstream}", fmt.Errorf("no upstream"))

	ops := NewGitOperations(repo, mockGit)

	branches, err := ops.ListBranches(false)

	require.NoError(t, err)
	assert.Len(t, branches, 3)
	assert.Equal(t, "main", branches[0].Name)
	assert.True(t, branches[0].Current)
	assert.Equal(t, "feature-1", branches[1].Name)
	assert.False(t, branches[1].Current)
	assert.Equal(t, "feature-2", branches[2].Name)
	assert.False(t, branches[2].Current)
}

func TestMergeBranch_Success(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("merge feature", "Merge made by the 'recursive' strategy.\n 1 file changed, 10 insertions(+)")
	mockGit.SetCommand("rev-parse HEAD", "newcommithash")
	mockGit.SetCommand("log -1 --pretty=format:%s", "Merge branch 'feature'")

	ops := NewGitOperations(repo, mockGit)

	result, err := ops.MergeBranch("feature", "main")

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.Equal(t, "newcommithash", result.CommitHash)
	assert.Equal(t, "Merge branch 'feature'", result.Message)
	assert.Equal(t, 1, result.FilesChanged)
}

func TestMergeBranch_EmptyBranches(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	_, err := ops.MergeBranch("", "main")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")

	_, err = ops.MergeBranch("feature", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must be specified")
}

func TestMergeBranch_SourceNotExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))

	ops := NewGitOperations(repo, mockGit)

	_, err := ops.MergeBranch("nonexistent", "main")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestMergeBranch_Conflict(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetError("merge feature", fmt.Errorf("CONFLICT (content): Merge conflict in file.txt"))

	ops := NewGitOperations(repo, mockGit)

	result, err := ops.MergeBranch("feature", "main")

	assert.Error(t, err)
	assert.False(t, result.Success)
	assert.Contains(t, result.Conflicts, "file.txt")
}

func TestCheckoutBranch_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("checkout feature", "Switched to branch 'feature'")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CheckoutBranch("feature")

	assert.NoError(t, err)
	assert.Equal(t, "feature", repo.CurrentBranch)
}

func TestCheckoutBranch_EmptyName(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.CheckoutBranch("")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestCheckoutBranch_NotExists(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetError("rev-parse --verify nonexistent", fmt.Errorf("unknown revision"))

	ops := NewGitOperations(repo, mockGit)

	err := ops.CheckoutBranch("nonexistent")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestPushBranch_Success(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("push origin main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PushBranch("main", "origin", false)

	assert.NoError(t, err)
}

func TestPushBranch_DefaultValues(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("push origin main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PushBranch("", "", false)

	assert.NoError(t, err)
}

func TestPushBranch_Force(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify main", "abc123def")
	mockGit.SetCommand("push --force origin main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PushBranch("main", "origin", true)

	assert.NoError(t, err)
}

func TestPushBranchWithUpstream(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "feature"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("push -u origin feature", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PushBranchWithUpstream("feature", "origin")

	assert.NoError(t, err)
}

func TestPullBranch_Success(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("pull origin main", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PullBranch("main", "origin")

	assert.NoError(t, err)
}

func TestPullBranch_DifferentBranch(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-parse --verify feature", "def456ghi")
	mockGit.SetCommand("checkout feature", "Switched to branch 'feature'")
	mockGit.SetCommand("pull origin feature", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PullBranch("feature", "origin")

	assert.NoError(t, err)
}

func TestFetchAll(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("fetch --all", "Fetching origin\nFetching upstream")

	ops := NewGitOperations(repo, mockGit)

	err := ops.FetchAll()

	assert.NoError(t, err)
}

func TestStashChanges_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash push -m work in progress", "Saved working directory and index state")

	ops := NewGitOperations(repo, mockGit)

	err := ops.StashChanges("work in progress")

	assert.NoError(t, err)
}

func TestStashChanges_NoMessage(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash push", "Saved working directory and index state")

	ops := NewGitOperations(repo, mockGit)

	err := ops.StashChanges("")

	assert.NoError(t, err)
}

func TestPopStash(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash pop", "On branch main: work in progress")

	ops := NewGitOperations(repo, mockGit)

	err := ops.PopStash()

	assert.NoError(t, err)
}

func TestApplyStash_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash apply stash@{1}", "On branch main: work in progress")

	ops := NewGitOperations(repo, mockGit)

	err := ops.ApplyStash("stash@{1}")

	assert.NoError(t, err)
}

func TestApplyStash_Default(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash apply stash@{0}", "On branch main: work in progress")

	ops := NewGitOperations(repo, mockGit)

	err := ops.ApplyStash("")

	assert.NoError(t, err)
}

func TestListStashes(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	stashOutput := `stash@{0}|WIP on main: work in progress|abc123def|1640995200
stash@{1}|On feature: another stash|def456ghi|1640995100`

	mockGit.SetCommand("stash list --pretty=format:%gd|%gs|%gD|%at", stashOutput)

	ops := NewGitOperations(repo, mockGit)

	stashes, err := ops.ListStashes()

	require.NoError(t, err)
	assert.Len(t, stashes, 2)
	assert.Equal(t, 0, stashes[0].Index)
	assert.Equal(t, "WIP on main: work in progress", stashes[0].Message)
	assert.Equal(t, 1, stashes[1].Index)
	assert.Equal(t, "On feature: another stash", stashes[1].Message)
}

func TestListStashes_Empty(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash list --pretty=format:%gd|%gs|%gD|%at", "")

	ops := NewGitOperations(repo, mockGit)

	stashes, err := ops.ListStashes()

	require.NoError(t, err)
	assert.Len(t, stashes, 0)
}

func TestDropStash(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("stash drop stash@{1}", "Dropped stash@{1}")

	ops := NewGitOperations(repo, mockGit)

	err := ops.DropStash("stash@{1}")

	assert.NoError(t, err)
}

func TestCreateCommit_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("add file1.txt file2.txt", "")
	mockGit.SetCommand("commit -m Add new features", "[main abc123def] Add new features")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateCommit("Add new features", []string{"file1.txt", "file2.txt"})

	assert.NoError(t, err)
}

func TestCreateCommit_EmptyMessage(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateCommit("", []string{"file1.txt"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestCreateCommit_NoFiles(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("commit -m Add new features", "[main abc123def] Add new features")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateCommit("Add new features", nil)

	assert.NoError(t, err)
}

func TestGetCommitHistory(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	historyOutput := `abc123def|John Doe|1640995200|Initial commit
def456ghi|Jane Doe|1640995300|Add feature`

	mockGit.SetCommand("log --pretty=format:%H|%an|%at|%s -n 10 main", historyOutput)
	mockGit.SetCommand("show --name-only --pretty=format: abc123def", "README.md\nfile1.txt")
	mockGit.SetCommand("show --name-only --pretty=format: def456ghi", "feature.txt")

	ops := NewGitOperations(repo, mockGit)

	commits, err := ops.GetCommitHistory("main", 10)

	require.NoError(t, err)
	assert.Len(t, commits, 2)
	assert.Equal(t, "abc123def", commits[0].Hash)
	assert.Equal(t, "John Doe", commits[0].Author)
	assert.Equal(t, "Initial commit", commits[0].Message)
	assert.Len(t, commits[0].Files, 2)
}

func TestGetCommitHistory_DefaultValues(t *testing.T) {
	repo := createTestRepository()
	repo.CurrentBranch = "main"
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("log --pretty=format:%H|%an|%at|%s -n 10 main", "")

	ops := NewGitOperations(repo, mockGit)

	commits, err := ops.GetCommitHistory("", 0)

	require.NoError(t, err)
	assert.Len(t, commits, 0)
}

func TestCreateTag_Success(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("tag -a v1.0.0 -m Release version 1.0.0", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateTag("v1.0.0", "Release version 1.0.0", "")

	assert.NoError(t, err)
}

func TestCreateTag_EmptyName(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()
	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateTag("", "message", "")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot be empty")
}

func TestCreateTag_NoMessage(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("tag v1.0.0", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateTag("v1.0.0", "", "")

	assert.NoError(t, err)
}

func TestCreateTag_WithCommit(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("tag -a v1.0.0 -m Release version 1.0.0 abc123def", "")

	ops := NewGitOperations(repo, mockGit)

	err := ops.CreateTag("v1.0.0", "Release version 1.0.0", "abc123def")

	assert.NoError(t, err)
}

func TestDeleteTag(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("tag -d v1.0.0", "Deleted tag 'v1.0.0'")

	ops := NewGitOperations(repo, mockGit)

	err := ops.DeleteTag("v1.0.0")

	assert.NoError(t, err)
}

func TestListTags(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	tagOutput := `v1.0.0|abc123def|Release version 1.0.0|1640995200|John Doe
v1.1.0|def456ghi|Bug fix release|1640995300|Jane Doe`

	mockGit.SetCommand("tag -l --format=%(refname:short)|%(objectname)|%(contents)|%(taggerdate:unix)|%(taggername)", tagOutput)

	ops := NewGitOperations(repo, mockGit)

	tags, err := ops.ListTags()

	require.NoError(t, err)
	assert.Len(t, tags, 2)
	assert.Equal(t, "v1.0.0", tags[0].Name)
	assert.Equal(t, "abc123def", tags[0].Hash)
	assert.Equal(t, "Release version 1.0.0", tags[0].Message)
	assert.Equal(t, "John Doe", tags[0].Tagger)
}

func TestGetStatus(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	statusOutput := ` M file1.txt
?? file2.txt
A  file3.txt`

	mockGit.SetCommand("status --porcelain", statusOutput)

	ops := NewGitOperations(repo, mockGit)

	status, err := ops.GetStatus()

	require.NoError(t, err)
	assert.Len(t, status, 3)
	assert.Equal(t, " M", status["file1.txt"])
	assert.Equal(t, "??", status["file2.txt"])
	assert.Equal(t, "A ", status["file3.txt"])
}

func TestIsClean_Clean(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("status --porcelain", "")

	ops := NewGitOperations(repo, mockGit)

	clean, err := ops.IsClean()

	require.NoError(t, err)
	assert.True(t, clean)
}

func TestIsClean_Dirty(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("status --porcelain", " M file1.txt")

	ops := NewGitOperations(repo, mockGit)

	clean, err := ops.IsClean()

	require.NoError(t, err)
	assert.False(t, clean)
}

func TestGetAheadBehindCounts(t *testing.T) {
	repo := createTestRepository()
	mockGit := NewMockGitCmd()

	mockGit.SetCommand("rev-list --left-right --count main...origin/main", "3\t2")

	ops := NewGitOperations(repo, mockGit)

	ahead, behind, err := ops.getAheadBehindCounts("main", "origin/main")

	require.NoError(t, err)
	assert.Equal(t, 3, ahead)
	assert.Equal(t, 2, behind)
}

func TestParseConflicts(t *testing.T) {
	repo := createTestRepository()
	ops := NewGitOperations(repo, nil)

	conflictOutput := `Auto-merging file1.txt
CONFLICT (content): Merge conflict in file1.txt
Auto-merging file2.txt
CONFLICT (content): Merge conflict in file2.txt
Automatic merge failed; fix conflicts and then commit the result.`

	conflicts := ops.parseConflicts(conflictOutput)

	assert.Len(t, conflicts, 2)
	assert.Contains(t, conflicts, "file1.txt")
	assert.Contains(t, conflicts, "file2.txt")
}

func TestParseFilesChanged(t *testing.T) {
	repo := createTestRepository()
	ops := NewGitOperations(repo, nil)

	mergeOutput := "Merge made by the 'recursive' strategy.\n 3 files changed, 15 insertions(+), 2 deletions(-)"

	filesChanged := ops.parseFilesChanged(mergeOutput)

	assert.Equal(t, 3, filesChanged)
}

func TestParseStashIndex(t *testing.T) {
	repo := createTestRepository()
	ops := NewGitOperations(repo, nil)

	assert.Equal(t, 0, ops.parseStashIndex("stash@{0}"))
	assert.Equal(t, 5, ops.parseStashIndex("stash@{5}"))
	assert.Equal(t, 0, ops.parseStashIndex("invalid"))
}