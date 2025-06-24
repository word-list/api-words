namespace WordList.Api.Words;

using System.Text.Json.Serialization;
using Amazon.Lambda.APIGatewayEvents;
using Microsoft.AspNetCore.Builder;
using Microsoft.Extensions.DependencyInjection;
using WordList.Data.Sql;
using Microsoft.AspNetCore.Http;
using Amazon.Lambda.Serialization.SystemTextJson;
using WordList.Api.Words.Models;
using WordList.Data.Sql.Models;

public class Function
{
    private static WordDb _wordDb = new();

    public static async Task Main(string[] args)
    {
        await WordAttributes.LoadAsync().ConfigureAwait(false);
        var attributes = (await WordAttributes.GetAllAsync().ConfigureAwait(false)).OrderBy(attr => attr.Name).ToList();

        var app = CreateHostBuilder(args).Build();
        var api = app.MapGroup("/api");

        api.MapGet("/words", new RequestDelegate(async (context) =>
        {
            string? GetQueryString(string name)
                => context.Request.Query.TryGetValue(name, out var value) ? value.ToString() : null;

            int? GetQueryInt(string name)
                => context.Request.Query.TryGetValue(name, out var value)
                    ? _ = int.TryParse(value, out int intValue)
                        ? intValue
                        : 0
                    : null;

            var queryAttributes = new Dictionary<string, AttributeRange>();

            foreach (var attr in attributes)
            {
                var minName = attr.Name + "Min";
                var maxName = attr.Name + "Max";

                int min = GetQueryInt(minName) ?? attr.Min;
                int max = GetQueryInt(maxName) ?? attr.Max;

                if (min != attr.Min || max != attr.Max)
                {
                    queryAttributes[attr.Name] = new(min, max);
                }
            }

            var text = GetQueryString("text");
            var from = GetQueryString("from");
            var randomSeed = GetQueryString("randomSeed");
            var randomCount = GetQueryInt("randomCount") ?? 0;
            var limit = GetQueryInt("limit") ?? 100;

            var words = await _wordDb.FindWordsAsync(text, null, queryAttributes, from, randomSeed, randomCount, limit).ConfigureAwait(false);

            var dtos = words
                .Select(w => new WordDto
                {
                    Text = w.Text,
                    Attributes = w.Attributes,
                    Types = []
                })
                .ToList();

            await context.Response.WriteAsJsonAsync(dtos, LambdaFunctionJsonSerializerContext.Default.ListWordDto).ConfigureAwait(false);
        }));

        await app.RunAsync();
    }

    public static WebApplicationBuilder CreateHostBuilder(string[] args)
    {
        var builder = WebApplication.CreateSlimBuilder(args);

        builder.Services.AddAWSLambdaHosting(LambdaEventSource.HttpApi, new SourceGeneratorLambdaJsonSerializer<LambdaFunctionJsonSerializerContext>());

        builder.Services.ConfigureHttpJsonOptions(options =>
        {
            options.SerializerOptions.TypeInfoResolverChain.Insert(0, LambdaFunctionJsonSerializerContext.Default);
        });

        return builder;
    }
}

[JsonSerializable(typeof(string))]
[JsonSerializable(typeof(List<string>))]
[JsonSerializable(typeof(Dictionary<string, int>))]
[JsonSerializable(typeof(WordDto))]
[JsonSerializable(typeof(List<WordDto>))]
[JsonSerializable(typeof(APIGatewayHttpApiV2ProxyRequest))]
[JsonSerializable(typeof(APIGatewayHttpApiV2ProxyResponse))]
public partial class LambdaFunctionJsonSerializerContext : JsonSerializerContext
{
}