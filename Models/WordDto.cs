namespace WordList.Api.Words.Models;

public class WordDto
{
    public required string Text { get; set; }

    public List<string> Types { get; set; } = [];

    public Dictionary<string, int> Attributes { get; set; } = [];
}